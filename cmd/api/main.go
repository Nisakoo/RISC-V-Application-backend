package main

import (
	"log"

	"riscv-lab/internal/app"
)

func main() {
	server := app.NewServer()

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
