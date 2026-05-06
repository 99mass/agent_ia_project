package config

import (
	"fmt"
	"log"
	"os"

	"github.com/unchk/agent-ia-backend/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable TimeZone=Africa/Dakar",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_USER", "unchk_admin"),
		getEnv("DB_PASSWORD", "unchk_password"),
		getEnv("DB_NAME", "unchk_agent_ia"),
	)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Successfully connected to the database")

	// Auto-migrate models
	err = database.AutoMigrate(
		&models.User{},
		&models.TestNiveau{},
		&models.Question{},
		&models.Reponse{},
		&models.Resultat{},
		&models.RessourcePedagogique{},
		&models.Recommandation{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Database migration completed")
	DB = database
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
