package main

import (
	"os"

	"github.com/vaishak/tf-migrate/cmd/tf-migrate/root"
)

func main() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}