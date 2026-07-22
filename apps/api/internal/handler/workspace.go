package handler

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/riffpad/riffpad/apps/api/internal/sandbox"
	"github.com/riffpad/riffpad/apps/api/internal/service"
)

type WorkspaceHandler struct {
	manager *service.WorkspaceManager
}

func NewWorkspaceHandler(manager *service.WorkspaceManager) *WorkspaceHandler {
	return &WorkspaceHandler{manager: manager}
}

func (h *WorkspaceHandler) Create(c echo.Context) error {
	// TODO: extract owner from JWT once auth is wired
	ownerID := "anonymous"
	ws, err := h.manager.Create(ownerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, ws)
}

func (h *WorkspaceHandler) List(c echo.Context) error {
	// TODO: extract owner from JWT once auth is wired
	ownerID := "anonymous"
	list, err := h.manager.List(ownerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, list)
}

func (h *WorkspaceHandler) Get(c echo.Context) error {
	id := c.Param("id")
	ws, ok := h.manager.Get(id)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "workspace not found"})
	}
	return c.JSON(http.StatusOK, ws)
}

func (h *WorkspaceHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	if _, ok := h.manager.Get(id); !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "workspace not found"})
	}
	if err := h.manager.Delete(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *WorkspaceHandler) ListMessages(c echo.Context) error {
	id := c.Param("id")
	if _, ok := h.manager.Get(id); !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "workspace not found"})
	}
	return c.JSON(http.StatusOK, h.manager.Messages(id))
}

func (h *WorkspaceHandler) ListFiles(c echo.Context) error {
	id := c.Param("id")
	dir := c.QueryParam("dir")
	if dir == "" {
		dir = "."
	}
	recursive := c.QueryParam("recursive") == "true"

	var files []sandbox.FileInfo
	var err error
	if recursive {
		files, err = h.manager.ListFilesRecursive(context.Background(), id, dir)
	} else {
		files, err = h.manager.ListFiles(context.Background(), id, dir)
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, files)
}

func (h *WorkspaceHandler) ReadFile(c echo.Context) error {
	id := c.Param("id")
	path := c.QueryParam("path")
	if path == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "path is required"})
	}
	content, err := h.manager.ReadFile(context.Background(), id, path)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"content": content})
}
