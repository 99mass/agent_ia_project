package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/unchk/agent-ia-backend/config"
	"github.com/unchk/agent-ia-backend/models"
	"golang.org/x/crypto/bcrypt"
)

// LoginPayload represents the expected login request body
type LoginPayload struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// ChangePasswordPayload represents the expected change password request body
type ChangePasswordPayload struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// Login authenticates a user and returns a JWT token
func Login(c *gin.Context) {
	var payload LoginPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Veuillez fournir un email et un mot de passe valides"})
		return
	}

	// Find user by email
	var user models.User
	if err := config.DB.Where("email = ?", payload.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants incorrects"})
		return
	}

	// Check if account is active
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Ce compte a été désactivé. Contactez l'administrateur."})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants incorrects"})
		return
	}

	// Generate JWT token
	token, err := config.GenerateToken(user.ID, user.Email, user.Role, user.MustChangePassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la génération du token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":                token,
		"must_change_password": user.MustChangePassword,
		"user": gin.H{
			"id":      user.ID,
			"nom":     user.Nom,
			"prenom":  user.Prenom,
			"email":   user.Email,
			"role":    user.Role,
			"eno":     user.ENO,
			"filiere": user.Filiere,
			"niveau":  user.Niveau,
		},
	})
}

// ChangePassword allows an authenticated user to change their password
func ChangePassword(c *gin.Context) {
	var payload ChangePasswordPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le nouveau mot de passe doit contenir au moins 8 caractères"})
		return
	}

	// Get current user from context
	userID, _ := c.Get("user_id")

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Mot de passe actuel incorrect"})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du hashage du mot de passe"})
		return
	}

	// Update password and set MustChangePassword to false
	user.Password = string(hashedPassword)
	user.MustChangePassword = false

	if err := config.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour du mot de passe"})
		return
	}

	// Generate a new token with updated claims
	newToken, err := config.GenerateToken(user.ID, user.Email, user.Role, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la génération du nouveau token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Mot de passe modifié avec succès",
		"token":   newToken,
		"user": gin.H{
			"id":      user.ID,
			"nom":     user.Nom,
			"prenom":  user.Prenom,
			"email":   user.Email,
			"role":    user.Role,
			"eno":     user.ENO,
			"filiere": user.Filiere,
			"niveau":  user.Niveau,
		},
	})
}
