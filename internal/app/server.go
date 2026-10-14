package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "riscv-lab/docs/swagger"
	"riscv-lab/internal/domain"
	"riscv-lab/internal/handler"
	"riscv-lab/internal/middleware"
	"riscv-lab/internal/route"
)

// API объединяет обработчики REST API и проверку JWT
type API struct {
	RiscVCommands *handler.RiscVAPIHandler
	Users         *handler.UserAPIHandler
	Tokens        middleware.TokenParser
}

// creatorRoles — роли, которым доступны методы создателя команд RISC-V
var creatorRoles = []domain.UserRole{domain.UserRoleUser, domain.UserRoleModerator}

type Server struct {
	router  *gin.Engine
	address string
}

func NewServer(address string, riscvCommandHandler *handler.RiscVCommandHandler, api *API) *Server {
	router := gin.Default()
	router.LoadHTMLGlob("templates/*.html")
	router.Static("/static", "resources")
	router.StaticFile("/defaults/image.svg", "assets/images/default.svg")
	router.StaticFile("/defaults/video.mp4", "assets/video/ADD.mp4")

	router.GET(route.Root, func(context *gin.Context) {
		context.Redirect(http.StatusFound, route.Feed)
	})
	router.GET(route.Feed, riscvCommandHandler.Feed)
	router.GET(route.Add, riscvCommandHandler.AddPage)
	router.GET(route.Grid, riscvCommandHandler.Grid)
	router.POST(route.Add, riscvCommandHandler.CreateDraft)
	router.POST(route.Publish, riscvCommandHandler.PublishDraft)
	router.POST(route.Delete, riscvCommandHandler.Delete)
	if api != nil {
		registerAPI(router, api)
	}

	return &Server{router: router, address: address}
}

func registerAPI(router *gin.Engine, api *API) {
	optional := middleware.OptionalAuth(api.Tokens)
	creator := middleware.RequireAuth(api.Tokens, creatorRoles...)

	commands := router.Group("/api/riscv_commands")
	commands.GET("", optional, api.RiscVCommands.List)
	commands.GET("/feed", optional, api.RiscVCommands.Feed)
	commands.GET("/draft", creator, api.RiscVCommands.Draft)
	commands.POST("", creator, api.RiscVCommands.Create)
	commands.PUT("/:id/publish", creator, api.RiscVCommands.Publish)
	commands.DELETE("/:id", creator, api.RiscVCommands.Delete)
	commands.POST("/:id/like", creator, api.RiscVCommands.Like)

	users := router.Group("/api/users")
	users.POST("", api.Users.Register)
	users.POST("/authentication", api.Users.Authenticate)
	users.POST("/logout", middleware.RequireAuth(api.Tokens), api.Users.Logout)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
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
