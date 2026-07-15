package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
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

func (h *WorkspaceHandler) Get(c echo.Context) error {
	id := c.Param("id")
	ws, ok := h.manager.Get(id)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "workspace not found"})
	}
	return c.JSON(http.StatusOK, ws)
}
