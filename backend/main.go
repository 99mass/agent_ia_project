package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/unchk/agent-ia-backend/config"
	"github.com/unchk/agent-ia-backend/routes"
	"github.com/unchk/agent-ia-backend/seed"
)

func main() {
	// Initialize JWT
	config.InitJWT()

	// Initialize Database
	config.ConnectDatabase()

	// Seed the database with mock UNCHK data
	seed.SeedDatabase(config.DB)

	// Initialize Gin Router
	r := gin.Default()

	// Setup all API routes
	routes.SetupRoutes(r)

	// Simple Ping Route
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Bienvenue sur l'API de recommandation UNCHK",
		})
	})

	// Run the server
	log.Println("Server is running on port 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
