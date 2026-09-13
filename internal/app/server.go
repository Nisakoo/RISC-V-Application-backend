package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"riscv-lab/internal/handler"
	"riscv-lab/internal/route"
)

type Server struct {
	router  *gin.Engine
	address string
}

func NewServer(address string, riscvCommandHandler *handler.RiscVCommandHandler) *Server {
	router := gin.Default()
	router.LoadHTMLGlob("templates/*.html")
	router.Static("/static", "resources")

	router.GET(route.Root, func(context *gin.Context) {
		context.Redirect(http.StatusFound, route.Feed)
	})
	router.GET(route.Feed, riscvCommandHandler.Feed)
	router.GET(route.Add, riscvCommandHandler.AddPage)
	router.GET(route.Grid, riscvCommandHandler.Grid)
	router.POST(route.Add, riscvCommandHandler.CreateDraft)
	router.POST(route.Publish, riscvCommandHandler.PublishDraft)
	router.POST(route.Delete, riscvCommandHandler.Delete)

	return &Server{router: router, address: address}
}

func (server *Server) Run(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:              server.address,
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
