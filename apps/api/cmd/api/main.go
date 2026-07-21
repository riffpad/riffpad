package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/riffpad/riffpad/apps/api/internal/config"
	"github.com/riffpad/riffpad/apps/api/internal/handler"
	"github.com/riffpad/riffpad/apps/api/internal/service"
)

func main() {
	// Try to load .env from common locations. The API is usually run from
	// apps/api during development, so look up to the project root first.
	loaded := false
	for _, path := range []string{"../../.env", ".env"} {
		if err := godotenv.Load(path); err == nil {
			loaded = true
			break
		}
	}
	if !loaded {
		log.Println("No .env file found, relying on environment variables")
	}
	cfg := config.Load()

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	workspaceManager := service.NewWorkspaceManager()

	h := handler.New()
	e.GET("/healthz", h.Healthz)

	wh := handler.NewWorkspaceHandler(workspaceManager)
	e.POST("/api/v1/workspaces", wh.Create)
	e.GET("/api/v1/workspaces/:id", wh.Get)
	e.GET("/api/v1/workspaces/:id/files", wh.ListFiles)
	e.GET("/api/v1/workspaces/:id/file", wh.ReadFile)

	ah := handler.NewAgentHandler(workspaceManager, cfg)
	e.GET("/ws/workspaces/:id", ah.Handle)

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	go func() {
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server: ", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	log.Println("server stopped")
}
