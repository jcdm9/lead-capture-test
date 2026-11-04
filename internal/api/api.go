package api

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"leadcapture/internal/importcsv"
	"leadcapture/internal/lead"
	"leadcapture/internal/store"

	"github.com/gin-gonic/gin"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	Store *store.Store
}

// Register mounts API routes on r.
func (h *Handler) Register(r gin.IRoutes) {
	r.GET("/api/leads", h.listLeads)
	r.POST("/api/leads", h.createLead)
	r.POST("/api/leads/import", h.importLeads)
	r.GET("/api/leads/:id", h.getLead)
	r.PUT("/api/leads/:id", h.updateLead)
	r.DELETE("/api/leads/:id", h.deleteLead)
}

func (h *Handler) listLeads(c *gin.Context) {
	leads, err := h.Store.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list leads"})
		return
	}
	c.JSON(http.StatusOK, leads)
}

type errorBody struct {
	Errors map[string]string `json:"errors"`
}

func (h *Handler) createLead(c *gin.Context) {
	in, ok := bindLeadInput(c)
	if !ok {
		return
	}

	created, err := h.Store.Insert(in)
	if err != nil {
		writeStoreError(c, err, "failed to create lead")
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) getLead(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	l, err := h.Store.Get(id)
	if err != nil {
		writeStoreError(c, err, "failed to get lead")
		return
	}
	c.JSON(http.StatusOK, l)
}

func (h *Handler) updateLead(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	in, ok := bindLeadInput(c)
	if !ok {
		return
	}

	updated, err := h.Store.Update(id, in)
	if err != nil {
		writeStoreError(c, err, "failed to update lead")
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) deleteLead(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.Store.Delete(id); err != nil {
		writeStoreError(c, err, "failed to delete lead")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) importLeads(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is required"})
		return
	}
	if !isCSVFilename(file.Filename) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .csv files are accepted"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read uploaded file"})
		return
	}
	defer src.Close()

	result, err := importcsv.Reader(src, h.Store, io.Discard)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func isCSVFilename(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".csv")
}

func bindLeadInput(c *gin.Context) (lead.Input, bool) {
	var in lead.Input
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "malformed JSON"})
		return lead.Input{}, false
	}
	if errs := lead.Validate(&in); lead.HasErrors(errs) {
		c.JSON(http.StatusUnprocessableEntity, errorBody{Errors: errs})
		return lead.Input{}, false
	}
	return in, true
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lead id"})
		return 0, false
	}
	return id, true
}

func writeStoreError(c *gin.Context, err error, fallback string) {
	if errors.Is(err, store.ErrDuplicateEmail) {
		c.JSON(http.StatusUnprocessableEntity, errorBody{
			Errors: map[string]string{"email": "already exists"},
		})
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "lead not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": fallback})
}

// NewRouter builds a Gin engine with API routes (no static files).
func NewRouter(s *store.Store) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	h := &Handler{Store: s}
	h.Register(r)
	return r
}
