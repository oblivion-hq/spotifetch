package main

import (
	"context"
	"log"
	"os"
	"spotify-api/app"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var ctx = context.Background()

func main() {
	_ = godotenv.Load()

	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	app, err := app.InitApp(ctx)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	if err := app.Router.Run(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
