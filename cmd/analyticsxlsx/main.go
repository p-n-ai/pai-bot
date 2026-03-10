package main

import (
	"fmt"
	"os"

	"github.com/p-n-ai/pai-bot/internal/analyticsxlsx"
)

func main() {
	if err := analyticsxlsx.Run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
