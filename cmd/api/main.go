package main

import (
	"log"
	"os"
	"strings"

	"github.com/NursultanKoshoev11/SmetaCheck/internal/api"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("environment file was not loaded: %v", err)
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		api.RunProduction()
		return
	}
	api.RunHardened()
}
