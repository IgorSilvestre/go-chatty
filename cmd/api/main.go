package main

import (
	"context"
	"log"
	"net/http"
	"time"

	apiv1 "go-chatty/cmd/api/router/v1"
	"go-chatty/internal/infrastructure/database"
	queue "go-chatty/internal/infrastructure/queue/adapter"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

	// Connect to the database on startup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.NewPoolFromEnv(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "OK",
		})
	})

	apiv1.RegisterRoutes(r, pool)

	// Initialize Asynq server (worker) and launch in a goroutine
	srv, err := queue.NewAsynqServer()
	if err != nil {
		log.Fatalf("failed to initialize asynq server: %v", err)
	}
	go func() {
		if err := srv.Run(context.Background()); err != nil {
			log.Fatalf("asynq server error: %v", err)
		}
	}()

	// Start HTTP server (blocks until shutdown)
	_ = r.Run()
}
