package main

import (
	"log"

	"github.com/NursultanKoshoev11/SmetaCheck/internal/api"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("environment file was not loaded: %v", err)
	}
	api.RunHardened()
}
