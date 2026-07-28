// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
)

func main() {
	out := flag.String("out", "", "new env fragment path (required; existing paths are refused)")
	flag.Parse()
	if strings.TrimSpace(*out) == "" {
		fmt.Fprintln(os.Stderr, "-out is required")
		os.Exit(2)
	}
	if err := config.WriteNewSecretEnvFile(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Created private secret env file %s\n", *out)
}
