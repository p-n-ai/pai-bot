// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package apicontract

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type Route struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

func Collect(root string) ([]Route, error) {
	routes := make(map[Route]struct{})
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (selector.Sel.Name != "Handle" && selector.Sel.Name != "HandleFunc") {
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			pattern, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			method, path, ok := strings.Cut(pattern, " ")
			if !ok || !strings.HasPrefix(path, "/api/") || !isHTTPMethod(method) {
				return true
			}
			routes[Route{Method: method, Path: path}] = struct{}{}
			return true
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect API routes: %w", err)
	}

	result := make([]Route, 0, len(routes))
	for route := range routes {
		result = append(result, route)
	}
	slices.SortFunc(result, func(left, right Route) int {
		if byPath := strings.Compare(left.Path, right.Path); byPath != 0 {
			return byPath
		}
		return strings.Compare(left.Method, right.Method)
	})
	return result, nil
}

func Removed(base, head []Route) []Route {
	headRoutes := make(map[Route]struct{}, len(head))
	for _, route := range head {
		headRoutes[route] = struct{}{}
	}
	removed := make([]Route, 0)
	for _, route := range base {
		if _, exists := headRoutes[route]; !exists {
			removed = append(removed, route)
		}
	}
	slices.SortFunc(removed, func(left, right Route) int {
		if byPath := strings.Compare(left.Path, right.Path); byPath != 0 {
			return byPath
		}
		return strings.Compare(left.Method, right.Method)
	})
	return removed
}

func isHTTPMethod(value string) bool {
	switch value {
	case "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT":
		return true
	default:
		return false
	}
}
