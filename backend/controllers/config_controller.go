package controllers

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/unchk/agent-ia-backend/config"
	"github.com/unchk/agent-ia-backend/models"
)

// GetSystemConfigs retrieves all system configurations
func GetSystemConfigs(c *gin.Context) {
	var configs []models.SystemConfig
	if err := config.DB.Order("key ASC").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de charger les configurations"})
		return
	}
	c.JSON(http.StatusOK, configs)
}

// UpdateSystemConfigsPayload is the payload structure to update configs
type UpdateSystemConfigsPayload struct {
	Configs map[string]string `json:"configs" binding:"required"`
}

// UpdateSystemConfigs updates multiple system configuration entries
func UpdateSystemConfigs(c *gin.Context) {
	var payload UpdateSystemConfigsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format de données invalide"})
		return
	}

	tx := config.DB.Begin()
	for key, value := range payload.Configs {
		var cfg models.SystemConfig
		if err := tx.Where("key = ?", key).First(&cfg).Error; err != nil {
			// Configuration doesn't exist, create it
			cfg = models.SystemConfig{
				Key:         key,
				Value:       value,
				Description: "Configuration système personnalisée.",
				UpdatedAt:   time.Now(),
			}
			if err := tx.Create(&cfg).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Échec de l'enregistrement de " + key})
				return
			}
		} else {
			// Exist, update value
			cfg.Value = value
			cfg.UpdatedAt = time.Now()
			if err := tx.Save(&cfg).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Échec de la mise à jour de " + key})
				return
			}
		}
	}
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Configurations mises à jour avec succès"})
}

// ServiceStatus represents the health of a platform service
type ServiceStatus struct {
	Name      string `json:"name"`
	Details   string `json:"details"`
	IsOnline  bool   `json:"is_online"`
	Timestamp string `json:"timestamp"`
}

// GetSystemStatus checks the connection status of the DB and the Python AI agent
func GetSystemStatus(c *gin.Context) {
	var statusList []ServiceStatus

	// 1. PostgreSQL DB Ping
	dbOnline := false
	sqlDB, err := config.DB.DB()
	if err == nil {
		if errPing := sqlDB.Ping(); errPing == nil {
			dbOnline = true
		}
	}
	statusList = append(statusList, ServiceStatus{
		Name:      "PostgreSQL",
		Details:   "Base de données unchk_db sur port 5432",
		IsOnline:  dbOnline,
		Timestamp: time.Now().Format(time.RFC3339),
	})

	// 2. Go Backend (always true if we can execute this)
	statusList = append(statusList, ServiceStatus{
		Name:      "Backend Golang",
		Details:   "Serveur API REST sur port 8080",
		IsOnline:  true,
		Timestamp: time.Now().Format(time.RFC3339),
	})

	// 3. Python AI Agent Ping
	aiEngineURL := os.Getenv("AI_ENGINE_URL")
	if aiEngineURL == "" {
		aiEngineURL = "http://localhost:8000/api/recommend"
	}
	// We want to ping the base URL of the Python service
	aiBaseURL := strings.Replace(aiEngineURL, "/api/recommend", "", 1)
	if aiBaseURL == "" {
		aiBaseURL = "http://localhost:8000"
	}

	aiOnline := false
	client := http.Client{Timeout: 2 * time.Second}
	resp, errGet := client.Get(aiBaseURL + "/")
	if errGet == nil {
		// If it replies with any HTTP status (even 404 or 200), it's running
		aiOnline = true
		resp.Body.Close()
	}
	statusList = append(statusList, ServiceStatus{
		Name:      "Agent IA Python",
		Details:   "Service de recommandation intelligent sur port 8000",
		IsOnline:  aiOnline,
		Timestamp: time.Now().Format(time.RFC3339),
	})

	// 4. Frontend Next.js (always assumed true since client is browsing it)
	statusList = append(statusList, ServiceStatus{
		Name:      "Frontend Next.js",
		Details:   "Application utilisateur sur port 3000",
		IsOnline:  true,
		Timestamp: time.Now().Format(time.RFC3339),
	})

	c.JSON(http.StatusOK, statusList)
}

