package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"riscv-lab/internal/app"
	"riscv-lab/internal/config"
	"riscv-lab/internal/handler"
	"riscv-lab/internal/repository"
	"riscv-lab/internal/service"
	"riscv-lab/internal/storage"
)

// @title                       API команд RISC-V
// @version                     1.0
// @description                 Веб-сервис каталога команд RISC-V: лента, черновики, публикация, лайки и пользователи. Без JWT доступно только чтение списка и ленты.
// @BasePath                    /
// @schemes                     http
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 JWT доступа в формате «Bearer <token>» из ответа POST /api/users/authentication.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	applicationConfig, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	database, err := repository.OpenPostgres(applicationConfig.DatabaseDSN)
	if err != nil {
		log.Fatal(err)
	}
	sqlDatabase, err := database.DB()
	if err != nil {
		log.Fatalf("получение подключения PostgreSQL: %v", err)
	}
	defer sqlDatabase.Close()
	pingContext, cancelPing := context.WithTimeout(ctx, 5*time.Second)
	defer cancelPing()
	if err := sqlDatabase.PingContext(pingContext); err != nil {
		log.Fatalf("проверка подключения PostgreSQL: %v", err)
	}

	mediaStorage, err := storage.NewMinIOStorage(
		applicationConfig.MinIOEndpoint,
		applicationConfig.MinIOAccessKey,
		applicationConfig.MinIOSecretKey,
		applicationConfig.MinIOBucket,
		applicationConfig.MinIOUseSSL,
		applicationConfig.MinIOPublicBaseURL,
	)
	if err != nil {
		log.Fatal(err)
	}
	riscvCommandRepository := repository.NewRiscVCommandRepository(database)
	riscvCommandService := service.NewRiscVCommandService(riscvCommandRepository)
	riscvCommandHandler := handler.NewRiscVCommandHandler(riscvCommandService, mediaStorage)
	redisContext, cancelRedis := context.WithTimeout(ctx, 5*time.Second)
	defer cancelRedis()
	tokenBlacklist, err := storage.NewRedisTokenBlacklist(redisContext, applicationConfig.RedisAddress, applicationConfig.RedisPassword)
	if err != nil {
		log.Fatal(err)
	}
	defer tokenBlacklist.Close()

	riscvAPIService := service.NewRiscVAPIService(riscvCommandRepository)
	userService := service.NewUserService(riscvCommandRepository, tokenBlacklist, applicationConfig.JWTSecret, applicationConfig.JWTTTL)
	server := app.NewServer(applicationConfig.ServerAddress, riscvCommandHandler, &app.API{
		RiscVCommands: handler.NewRiscVAPIHandler(riscvAPIService, mediaStorage),
		Users:         handler.NewUserAPIHandler(userService),
		Tokens:        userService,
	})

	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
