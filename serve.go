package main

import (
	"embed"
	"fmt"
	"io/fs"

	"leadcapture/internal/server"
	"leadcapture/internal/store"
)

//go:embed all:web/dist
var embeddedDist embed.FS

func runServe() error {
	s, err := store.Open(defaultDB)
	if err != nil {
		return err
	}
	defer s.Close()

	dist, err := fs.Sub(embeddedDist, "web/dist")
	if err != nil {
		return fmt.Errorf("embed dist: %w", err)
	}

	fmt.Println("listening on http://localhost:8080")
	return server.Run(":8080", s, dist)
}
