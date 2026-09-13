package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"riscv-lab/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := app.NewServer()

	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
