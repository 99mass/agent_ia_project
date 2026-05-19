package routes

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/unchk/agent-ia-backend/controllers"
	"github.com/unchk/agent-ia-backend/middleware"
)

// SetupRoutes initializes all the routes for the application
func SetupRoutes(router *gin.Engine) {
	// CORS configuration for Next.js frontend
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	api := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/login", controllers.Login)
			auth.POST("/change-password", middleware.AuthMiddleware(), controllers.ChangePassword)
		}

		// Admin routes (protected: auth + admin role required)
		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware())
		{
			admin.GET("/users", controllers.GetUsers)
			admin.POST("/users", controllers.CreateUser)
			admin.PUT("/users/:id", controllers.UpdateUser)
			admin.DELETE("/users/:id", controllers.DeleteUser)
			admin.POST("/users/:id/reset-password", controllers.ResetUserPassword)
			admin.POST("/users/import-csv", controllers.ImportUsersCSV)
		}

		// Tests routes (protected: auth required)
		tests := api.Group("/tests")
		tests.Use(middleware.AuthMiddleware())
		{
			tests.GET("/", controllers.GetTests)
			tests.POST("/", controllers.CreateTest)
			tests.PUT("/:id", controllers.UpdateTest)
			tests.DELETE("/:id", controllers.DeleteTest)
			tests.POST("/import-csv", controllers.ImportTestsCSV)
			tests.POST("/:id/submit", controllers.SubmitTest)
		}

		// Resources routes (protected: auth required)
		resources := api.Group("/resources")
		resources.Use(middleware.AuthMiddleware())
		{
			resources.GET("/", controllers.GetResources)
			resources.POST("/", controllers.CreateResource)
			resources.PUT("/:id", controllers.UpdateResource)
			resources.DELETE("/:id", controllers.DeleteResource)
		}
	}
}
