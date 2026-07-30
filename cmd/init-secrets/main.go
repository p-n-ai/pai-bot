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
		if _, err := fmt.Fprintln(stderr, "-out is required"); err != nil {
			return 1
		}
		return 2
	}
	if err := config.WriteNewSecretEnvFile(*out); err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return 1
		}
		return 1
	}
	if _, err := fmt.Fprintf(stdout, "Created private secret env file %s\n", *out); err != nil {
		return 1
	}
	return 0
}
