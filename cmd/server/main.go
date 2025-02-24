package main

import (
	"log"
	"os"

	"github.com/antiartificial/baggins/internal/api"
	"github.com/antiartificial/baggins/internal/processor"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// Create storage directories
	dirs := []string{"uploads", "processed"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Initialize the processor
	proc := processor.NewMediaProcessor()

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		BodyLimit: 50 * 1024 * 1024, // 50MB limit
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	// Initialize API routes
	api.SetupRoutes(app, proc)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(app.Listen(":" + port))
}
