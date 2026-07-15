package handler

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/riffpad/riffpad/apps/api/internal/agent"
	"github.com/riffpad/riffpad/apps/api/internal/config"
	"github.com/riffpad/riffpad/apps/api/internal/service"
	"github.com/sashabaranov/go-openai"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type AgentHandler struct {
	manager *service.WorkspaceManager
	cfg     config.Config
	client  *openai.Client
}

func NewAgentHandler(manager *service.WorkspaceManager, cfg config.Config) *AgentHandler {
	apiKey := cfg.LLM.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}
	oc := openai.DefaultConfig(apiKey)
	oc.BaseURL = cfg.LLM.BaseURL
	return &AgentHandler{
		manager: manager,
		cfg:     cfg,
		client:  openai.NewClientWithConfig(oc),
	}
}

type wsMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

func (h *AgentHandler) Handle(c echo.Context) error {
	workspaceID := c.Param("id")
	ws, ok := h.manager.Get(workspaceID)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "workspace not found"})
	}

	box, ok := h.manager.Sandbox(workspaceID)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "sandbox not found"})
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	emitter := func(event agent.AgentEvent) error {
		mu.Lock()
		defer mu.Unlock()
		return conn.WriteJSON(event)
	}

	orch := agent.NewOrchestrator(h.client, h.cfg.LLM.Model, agent.PermissionConfirm)

	// Send initial workspace info
	_ = emitter(agent.AgentEvent{
		Type:      "workspace_connected",
		Content:   ws.Slug,
		Timestamp: time.Now().UnixMilli(),
	})

	// Read loop
	go func() {
		for {
			var msg wsMessage
			if err := conn.ReadJSON(&msg); err != nil {
				cancel()
				return
			}
			if msg.Type == "prompt" {
				go func(content string) {
					if err := orch.Run(ctx, box, content, emitter); err != nil {
						_ = emitter(agent.AgentEvent{
							Type:      "error",
							Content:   err.Error(),
							Timestamp: time.Now().UnixMilli(),
						})
					}
				}(msg.Content)
			}
		}
	}()

	// Wait for cancellation
	<-ctx.Done()
	return nil
}
