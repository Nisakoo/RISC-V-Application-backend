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
	riscvCommandService := service.NewRiscVCommandService(riscvCommandRepository, mediaStorage)
	riscvCommandHandler := handler.NewRiscVCommandHandler(riscvCommandService, mediaStorage)
	server := app.NewServer(applicationConfig.ServerAddress, riscvCommandHandler)

	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
