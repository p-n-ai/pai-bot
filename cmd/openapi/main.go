// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/p-n-ai/pai-bot/internal/apidocs"
)

func main() {
	document, err := apidocs.JSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate OpenAPI document: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(append(document, '\n')); err != nil {
		fmt.Fprintf(os.Stderr, "write OpenAPI document: %v\n", err)
		os.Exit(1)
	}
}
