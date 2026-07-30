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
		fail("-base and -head must be provided together")
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
		fail("%v", err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(routes); err != nil {
		fail("encode API routes: %v", err)
	}
}

func readRoutes(path string) []apicontract.Route {
	data, err := os.ReadFile(path)
	if err != nil {
		fail("read %s: %v", path, err)
	}

	var routes []apicontract.Route
	if err := json.Unmarshal(data, &routes); err != nil {
		fail("decode %s: %v", path, err)
	}
	return routes
}

func fail(format string, arguments ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", arguments...)
	os.Exit(1)
}
