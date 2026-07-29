// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
)

func main() {
	if err := config.ValidateProductionSecretEnvironment(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Production secrets are valid")
}
