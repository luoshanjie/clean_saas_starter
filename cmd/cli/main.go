package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "new-project" {
		if err := runNewProjectCommand(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "new-module" {
		if err := runNewModuleCommand(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/cli new-project --name <project_name> --output <path>")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/cli new-module --name <module_name>")
	os.Exit(1)
}
