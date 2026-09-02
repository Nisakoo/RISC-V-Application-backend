package app

import (
	"github.com/gin-gonic/gin"

	"riscv-lab/internal/handler"
	"riscv-lab/internal/repository"
)

type Server struct {
	router *gin.Engine
}

func NewServer() *Server {
	repository := repository.NewInstructionRepository()
	handler := handler.NewInstructionHandler(repository)
	router := gin.Default()

	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "resources")

	router.GET("/feed", handler.Feed)
	router.GET("/feed/:id", handler.Feed)
	router.GET("/instruction/draft", handler.Draft)
	router.GET("/instructions", handler.Grid)

	return &Server{router: router}
}

func (server *Server) Run() error {
	return server.router.Run(":8080")
}
