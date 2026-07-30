// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("init-secrets", flag.ContinueOnError)
	flags.SetOutput(stderr)
	out := flags.String("out", "", "new env fragment path (required; existing paths are refused)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*out) == "" {
		fmt.Fprintln(stderr, "-out is required")
		return 2
	}
	if err := config.WriteNewSecretEnvFile(*out); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Created private secret env file %s\n", *out)
	return 0
}
