package main

import (
	"context"
	chatTask "go-chatty/internal/pkg/chat/application/task"
	"log"
	"net/http"
	"time"

	apiv1 "go-chatty/cmd/api/router/v1"
	"go-chatty/internal/infrastructure/database"
	queueAdapter "go-chatty/internal/infrastructure/queue/adapter"
	queueport "go-chatty/internal/infrastructure/queue/port"
	"go-chatty/internal/infrastructure/realtime"

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

	// Initialize queue client (for producers)
	var qClient queueport.Client
	qClient, err = queueAdapter.NewAsynqClientFromEnv()
	if err != nil {
		log.Fatalf("failed to initialize asynq client: %v", err)
	}
	defer func() { _ = qClient.Close() }()

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "OK",
		})
	})

	// Router manages websocket fan-out per user/session
	realtimeRouter := realtime.NewRouter()
	defer realtimeRouter.Close()

	apiv1.RegisterRoutes(r, pool, qClient, realtimeRouter)

	// Initialize Asynq server (worker) and launch in a goroutine
	srv, err := queueAdapter.NewAsynqServer()
	if err != nil {
		log.Fatalf("failed to initialize asynq server: %v", err)
	}

	// Register chat tasks
	chatTask.RegisterSendMessageTask(srv, pool)

	go func() {
		if err := srv.Run(context.Background()); err != nil {
			log.Fatalf("asynq server error: %v", err)
		}
	}()

	// Start HTTP server (blocks until shutdown)
	_ = r.Run()
}
