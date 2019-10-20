package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/alexeldeib/incendiary-iguana/cmd/ensure"
)

var version string

func main() {
	if err := NewRootCommand(version).Execute(); err != nil {
		fmt.Printf("%+#v\n", err)
		os.Exit(1)
	}
}

func NewRootCommand(version string) *cobra.Command {
	root := &cobra.Command{
		Use: "tinker",
	}
	root.AddCommand(NewVersionCommand(version))
	root.AddCommand(ensure.NewEnsureCommand())
	root.AddCommand(ensure.NewDeleteCommand())
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
