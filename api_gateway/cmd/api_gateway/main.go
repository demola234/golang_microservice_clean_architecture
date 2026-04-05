package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/demola234/api_gateway/config"
	"github.com/demola234/api_gateway/infrastructure/grpc_clients"
	"github.com/demola234/api_gateway/infrastructure/middleware"
	"github.com/demola234/api_gateway/infrastructure/middleware/token_maker"
	"github.com/demola234/api_gateway/internal/handler"
	routes "github.com/demola234/api_gateway/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	configs, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Failed to load env file: %v", err)
	}

	// Initialize gRPC client with dynamic address
	authClient, err := grpc_clients.NewAuthenticationClient(configs.AuthenticationGRPCAddress, 20*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to Authentication service: %v", err)
	}
	defer authClient.Close()

	propertyClient, err := grpc_clients.NewPropertyClient(configs.PropertyGRPCAddress, 20*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to Property service: %v", err)
	}

	// messageClient, err := grpc_clients.NewMessagingClient("127.0.0.1:9093", 20*time.Second)
	// if err != nil {
	// 	log.Fatalf("Failed to connect to Property service: %v", err)
	// }

	// Create a new Gin router
	router := gin.Default()

	tokenMaker, err := token_maker.NewTokenMaker(configs.TokenSymmetricKey)
	if err != nil {
		panic("failed to create token maker: " + err.Error())
	}

	authMiddleware := middleware.AuthMiddleware(tokenMaker)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authClient)
	propertyHandler := handler.NewPropertyHandler(propertyClient)
	// messageHandler := handler.NewMessageHandler(messageClient)

	// Group routes under /v1
	v1 := router.Group("/v1")
	{
		// Health check endpoint
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "OK",
			})
		})
	}

	// Define authentication routes
	routes.RegisterRoutes(v1, authHandler, authMiddleware)
	routes.RegisterPropertyRoutes(v1, propertyHandler, authMiddleware)
	// routes.RegisterMessageRoutes(v1, messageHandler, authMiddleware)

	// Create an HTTP server with the configured port
	srv := &http.Server{
		Addr:    configs.Port,
		Handler: router,
	}

	// Start the server in a goroutine using tcp4 to ensure IPv4 binding (required for Android emulator)
	ln, err := net.Listen("tcp4", configs.Port)
	if err != nil {
		log.Fatalf("Could not listen on %s: %v", configs.Port, err)
	}
	go func() {
		log.Printf("Starting API Gateway at %s...", configs.Port)
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start server: %v", err)
		}
	}()

	// Wait for an interrupt signal to shut down gracefully
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down API Gateway...")

	// Create a timeout context for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("API Gateway stopped gracefully.")
}
