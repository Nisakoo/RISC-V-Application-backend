package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"riscv-lab/internal/handler"
	"riscv-lab/internal/repository"
	"riscv-lab/internal/service"
)

type Server struct {
	router *gin.Engine
}

func NewServer() *Server {
	repository := repository.NewInstructionRepository()
	instructionService := service.NewInstructionService(repository)
	handler := handler.NewInstructionHandler(instructionService)
	router := gin.Default()

	router.LoadHTMLGlob("templates/*.html")
	router.Static("/static", "resources")

	router.GET("/", handler.Feed)
	router.GET("/instruction/draft", handler.Draft)
	router.GET("/instructions", handler.Grid)

	return &Server{router: router}
}

func (server *Server) Run(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           server.router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveError := make(chan error, 1)
	go func() {
		serveError <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serveError:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownContext)
	}
}
