package main

import (
	"fmt"
	"os"

	"leadcapture/internal/importcsv"
	"leadcapture/internal/store"
)

const defaultDB = "leads.db"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "import":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: leads import <file.csv>")
			os.Exit(2)
		}
		if err := runImport(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "serve":
		if err := runServe(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: leads <import|serve> ...")
}

func runImport(path string) error {
	s, err := store.Open(defaultDB)
	if err != nil {
		return err
	}
	defer s.Close()

	result, err := importcsv.File(path, s, os.Stderr)
	if err != nil {
		return err
	}
	fmt.Printf("imported %d, skipped %d\n", result.Imported, result.Skipped)
	return nil
}

// runServe is implemented in serve.go (embed + HTTP).
