// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/p-n-ai/pai-bot/internal/apicontract"
)

func main() {
	root := flag.String("root", "internal/server", "source directory containing API route registrations")
	basePath := flag.String("base", "", "base route contract to compare")
	headPath := flag.String("head", "", "head route contract to compare")
	flag.Parse()

	if (*basePath == "") != (*headPath == "") {
		failMessage("-base and -head must be provided together")
	}
	if *basePath != "" {
		base := readRoutes(*basePath)
		head := readRoutes(*headPath)
		removed := apicontract.Removed(base, head)
		if len(removed) == 0 {
			return
		}
		for _, route := range removed {
			fmt.Fprintf(os.Stderr, "removed runtime API route: %s %s\n", route.Method, route.Path)
		}
		os.Exit(1)
	}

	routes, err := apicontract.Collect(*root)
	if err != nil {
		failError(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(routes); err != nil {
		failError(fmt.Errorf("encode API routes: %w", err))
	}
}

func readRoutes(path string) []apicontract.Route {
	data, err := os.ReadFile(path)
	if err != nil {
		failError(fmt.Errorf("read %s: %w", path, err))
	}

	var routes []apicontract.Route
	if err := json.Unmarshal(data, &routes); err != nil {
		failError(fmt.Errorf("decode %s: %w", path, err))
	}
	return routes
}

func failMessage(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

func failError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
