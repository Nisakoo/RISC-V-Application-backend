package main

import (
	"context"
	"log"
	"os"

	"riscv-lab/internal/config"
	"riscv-lab/internal/migration"
	"riscv-lab/internal/repository"
)

func main() {
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

	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}
	switch direction {
	case "up":
		err = migration.Up(context.Background(), database)
	case "down":
		err = migration.Down(context.Background(), database)
	default:
		log.Fatalf("неизвестное направление миграции %q: используйте up или down", direction)
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("миграция %s выполнена", direction)
}
