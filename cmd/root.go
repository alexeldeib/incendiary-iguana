package main

import (
	"fmt"
	"os"

	"github.com/google/wire"
	"github.com/spf13/cobra"
)

func NewRootCommand(version string) *cobra.Command {
	wire.Build(NewVersionCommand, NewEnsureCommand)
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

func NewEnsureCommand() *cobra.Command {
	opts := NewEnsureOptions()
	cmd := &cobra.Command{
		Use:   "ensure",
		Short: "Ensure reconciles actual resource state to match desired",
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Ensure(); err != nil {
				fmt.Printf("%+#v\n", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVarP(&opts.File, "file", "f", "", "File containt one or more Kubernetes manifests from a file containing multiple YAML documents (---)")
	cmd.MarkFlagRequired("file")
	return cmd
}
