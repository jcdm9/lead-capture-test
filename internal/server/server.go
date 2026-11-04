package server

import (
	"io/fs"
	"net/http"
	"strings"

	"leadcapture/internal/api"
	"leadcapture/internal/store"

	"github.com/gin-gonic/gin"
)

// Run starts the HTTP server on addr, serving the API and static frontend from dist.
func Run(addr string, s *store.Store, dist fs.FS) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	h := &api.Handler{Store: s}
	h.Register(r)

	static, err := fs.Sub(dist, ".")
	if err != nil {
		return err
	}
	fileServer := http.FileServer(http.FS(static))

	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if f, err := static.Open(path); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		// SPA fallback
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	return r.Run(addr)
}
