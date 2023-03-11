package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sanity",
	Short: "a wrapper for arxiv-sanity-lite, built with Go",
}

func init() {
	rootCmd.AddCommand(
		serveCmd,
		pollCmd,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
}
