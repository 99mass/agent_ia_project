package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/unchk/agent-ia-backend/config"
	"github.com/unchk/agent-ia-backend/models"
)

// GetConcepts returns all concepts sorted alphabetically
func GetConceptsList(c *gin.Context) {
	var concepts []models.Concept
	if err := config.DB.Order("nom ASC").Find(&concepts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des concepts"})
		return
	}
	c.JSON(http.StatusOK, concepts)
}

// CreateConceptPayload represents the request body
type CreateConceptPayload struct {
	Nom string `json:"nom" binding:"required"`
}

// CreateConcept adds a new concept
func CreateConcept(c *gin.Context) {
	var payload CreateConceptPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le nom du concept est requis"})
		return
	}

	nom := strings.TrimSpace(payload.Nom)
	if nom == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le nom du concept ne peut pas être vide"})
		return
	}

	concept := models.Concept{Nom: nom}

	// Check if already exists
	var existing models.Concept
	if err := config.DB.Where("LOWER(nom) = LOWER(?)", nom).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Ce concept existe déjà", "concept": existing})
		return
	}

	if err := config.DB.Create(&concept).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du concept"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Concept créé avec succès", "concept": concept})
}

// DeleteConcept removes a concept (Admin only)
func DeleteConcept(c *gin.Context) {
	// Authorization check
	userRole, exists := c.Get("user_role")
	if !exists || userRole.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Seuls les administrateurs peuvent supprimer des concepts"})
		return
	}

	id := c.Param("id")
	var concept models.Concept

	if err := config.DB.First(&concept, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Concept non trouvé"})
		return
	}

	if err := config.DB.Delete(&concept).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Concept supprimé avec succès"})
}

// UpdateConcept des-updates a concept's name
func UpdateConcept(c *gin.Context) {
	id := c.Param("id")
	var concept models.Concept

	if err := config.DB.First(&concept, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Concept non trouvé"})
		return
	}

	var payload CreateConceptPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le nom du concept est requis"})
		return
	}

	nom := strings.TrimSpace(payload.Nom)
	if nom == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le nom du concept ne peut pas être vide"})
		return
	}

	// Check if another concept has the same name
	var existing models.Concept
	if err := config.DB.Where("LOWER(nom) = LOWER(?) AND id != ?", nom, concept.ID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Ce concept existe déjà"})
		return
	}

	concept.Nom = nom
	if err := config.DB.Save(&concept).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la modification du concept"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Concept modifié avec succès", "concept": concept})
}
