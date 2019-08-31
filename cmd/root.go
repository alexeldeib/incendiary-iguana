package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRootCommand(version string) *cobra.Command {
	root := &cobra.Command{
		Use: "tinker",
	}
	root.AddCommand(NewVersionCommand(version))
	root.AddCommand(NewEnsureCommand())
	return root
}

func NewVersionCommand(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Version: %s\n", version)
		},
	}
	return cmd
}
