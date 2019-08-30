package main

import (
	"fmt"
	"os"
)

var version string

func main() {
	if err := NewRootCommand(version).Execute(); err != nil {
		fmt.Printf("%+#v\n", err)
		os.Exit(1)
	}
}
