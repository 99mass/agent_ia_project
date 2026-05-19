package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/unchk/agent-ia-backend/config"
	"github.com/unchk/agent-ia-backend/models"
)

// GetResources returns a list of pedagogical resources.
// It can optionally filter by auteur_id if specified.
func GetResources(c *gin.Context) {
	var resources []models.RessourcePedagogique
	auteurID := c.Query("auteur_id")

	query := config.DB.Model(&models.RessourcePedagogique{})
	if auteurID != "" {
		query = query.Where("auteur_id = ?", auteurID)
	}

	if err := query.Order("created_at DESC").Find(&resources).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des ressources"})
		return
	}

	c.JSON(http.StatusOK, resources)
}

// CreateResourcePayload represents the request body for creating a resource
type CreateResourcePayload struct {
	Titre            string `json:"titre" binding:"required"`
	Type             string `json:"type" binding:"required"`
	Lien             string `json:"lien" binding:"required"`
	Concept          string `json:"concept" binding:"required"`
	NiveauDifficulte int    `json:"niveau_difficulte" binding:"required"`
	Pertinence       int    `json:"pertinence"`
}

// CreateResource adds a new pedagogical resource
func CreateResource(c *gin.Context) {
	var payload CreateResourcePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	// Get auteur_id from token
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non autorisé"})
		return
	}

	resource := models.RessourcePedagogique{
		Titre:            payload.Titre,
		Type:             payload.Type,
		Lien:             payload.Lien,
		Concept:          payload.Concept,
		NiveauDifficulte: payload.NiveauDifficulte,
		Pertinence:       payload.Pertinence,
		AuteurID:         uint(userID.(float64)),
	}

	if err := config.DB.Create(&resource).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de la ressource"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Ressource créée avec succès", "ressource": resource})
}

// UpdateResource modifies an existing resource
func UpdateResource(c *gin.Context) {
	id := c.Param("id")
	var resource models.RessourcePedagogique

	if err := config.DB.First(&resource, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ressource non trouvée"})
		return
	}

	// Authorization check
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")
	if userRole != "admin" && uint(userID.(float64)) != resource.AuteurID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Non autorisé à modifier cette ressource"})
		return
	}

	var payload CreateResourcePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	resource.Titre = payload.Titre
	resource.Type = payload.Type
	resource.Lien = payload.Lien
	resource.Concept = payload.Concept
	resource.NiveauDifficulte = payload.NiveauDifficulte
	resource.Pertinence = payload.Pertinence

	if err := config.DB.Save(&resource).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ressource mise à jour avec succès", "ressource": resource})
}

// DeleteResource deletes an existing resource
func DeleteResource(c *gin.Context) {
	id := c.Param("id")
	var resource models.RessourcePedagogique

	if err := config.DB.First(&resource, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ressource non trouvée"})
		return
	}

	// Authorization check
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")
	if userRole != "admin" && uint(userID.(float64)) != resource.AuteurID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Non autorisé à supprimer cette ressource"})
		return
	}

	if err := config.DB.Delete(&resource).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ressource supprimée avec succès"})
}
