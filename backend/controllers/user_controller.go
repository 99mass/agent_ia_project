package controllers

import (
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/unchk/agent-ia-backend/config"
	"github.com/unchk/agent-ia-backend/models"
	"golang.org/x/crypto/bcrypt"
)

// CreateUserPayload represents the expected request body for creating a user
type CreateUserPayload struct {
	Nom     string `json:"nom" binding:"required"`
	Prenom  string `json:"prenom" binding:"required"`
	Email   string `json:"email" binding:"required,email"`
	Role    string `json:"role" binding:"required"`
	ENO     string `json:"eno"`
	Filiere string `json:"filiere"`
	Niveau  string `json:"niveau"`
}

// UpdateUserPayload represents the expected request body for updating a user
type UpdateUserPayload struct {
	Nom      string `json:"nom"`
	Prenom   string `json:"prenom"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	ENO      string `json:"eno"`
	Filiere  string `json:"filiere"`
	Niveau   string `json:"niveau"`
	IsActive *bool  `json:"is_active"`
}

// GetUsers returns all users (admin only)
func GetUsers(c *gin.Context) {
	var users []models.User
	config.DB.Order("created_at DESC").Find(&users)
	c.JSON(http.StatusOK, users)
}

// CreateUser creates a new user with a temporary password (admin only)
func CreateUser(c *gin.Context) {
	var payload CreateUserPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides : nom, prénom, email et rôle sont requis"})
		return
	}

	// Validate role
	if payload.Role != "etudiant" && payload.Role != "enseignant" && payload.Role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rôle invalide. Valeurs acceptées : etudiant, enseignant, admin"})
		return
	}

	// Check if email already exists
	var existing models.User
	if err := config.DB.Where("email = ?", payload.Email).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Un utilisateur avec cet email existe déjà"})
		return
	}

	// Generate temporary password
	tempPassword := generateTempPassword(8)

	// Hash the temporary password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du mot de passe"})
		return
	}

	user := models.User{
		Nom:                payload.Nom,
		Prenom:             payload.Prenom,
		Email:              payload.Email,
		Password:           string(hashedPassword),
		Role:               payload.Role,
		IsActive:           true,
		MustChangePassword: true,
		ENO:                payload.ENO,
		Filiere:            payload.Filiere,
		Niveau:             payload.Niveau,
	}

	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de l'utilisateur"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Utilisateur créé avec succès",
		"user":           user,
		"temp_password":  tempPassword,
	})
}

// UpdateUser updates an existing user (admin only)
func UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User

	if err := config.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
		return
	}

	var payload UpdateUserPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	// Update fields if provided
	if payload.Nom != "" {
		user.Nom = payload.Nom
	}
	if payload.Prenom != "" {
		user.Prenom = payload.Prenom
	}
	if payload.Email != "" {
		// Check if email is taken by another user
		var existing models.User
		if err := config.DB.Where("email = ? AND id != ?", payload.Email, user.ID).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Cet email est déjà utilisé par un autre utilisateur"})
			return
		}
		user.Email = payload.Email
	}
	if payload.Role != "" {
		if payload.Role != "etudiant" && payload.Role != "enseignant" && payload.Role != "admin" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Rôle invalide"})
			return
		}
		user.Role = payload.Role
	}
	if payload.ENO != "" {
		user.ENO = payload.ENO
	}
	if payload.Filiere != "" {
		user.Filiere = payload.Filiere
	}
	if payload.Niveau != "" {
		user.Niveau = payload.Niveau
	}
	if payload.IsActive != nil {
		user.IsActive = *payload.IsActive
	}

	if err := config.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Utilisateur mis à jour avec succès",
		"user":    user,
	})
}

// DeleteUser soft-deletes a user (admin only)
func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User

	if err := config.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
		return
	}

	if err := config.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Utilisateur supprimé avec succès"})
}

// ResetUserPassword resets a user's password to a new temporary one (admin only)
func ResetUserPassword(c *gin.Context) {
	id := c.Param("id")
	var user models.User

	if err := config.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
		return
	}

	// Generate new temporary password
	tempPassword := generateTempPassword(8)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la réinitialisation"})
		return
	}

	user.Password = string(hashedPassword)
	user.MustChangePassword = true

	if err := config.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Mot de passe réinitialisé avec succès",
		"temp_password": tempPassword,
	})
}

// ImportUsersCSV handles bulk import of users from a CSV file (admin only)
// CSV format: nom,prenom,email,role,eno,filiere,niveau
func ImportUsersCSV(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fichier CSV requis"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header row
	header, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Impossible de lire l'en-tête du fichier CSV"})
		return
	}

	// Map header columns to indices
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(strings.ToLower(col))] = i
	}

	// Validate required columns
	requiredCols := []string{"nom", "prenom", "email", "role"}
	for _, col := range requiredCols {
		if _, exists := colMap[col]; !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Colonne requise manquante : %s. Colonnes attendues : nom,prenom,email,role,eno,filiere,niveau", col),
			})
			return
		}
	}

	type ImportResult struct {
		Nom          string `json:"nom"`
		Prenom       string `json:"prenom"`
		Email        string `json:"email"`
		Role         string `json:"role"`
		TempPassword string `json:"temp_password"`
		Status       string `json:"status"`
		Error        string `json:"error,omitempty"`
	}

	var results []ImportResult
	successCount := 0
	errorCount := 0
	lineNumber := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		lineNumber++

		if err != nil {
			results = append(results, ImportResult{
				Status: "erreur",
				Error:  fmt.Sprintf("Ligne %d : erreur de lecture", lineNumber),
			})
			errorCount++
			continue
		}

		// Extract fields
		nom := getCSVField(record, colMap, "nom")
		prenom := getCSVField(record, colMap, "prenom")
		email := getCSVField(record, colMap, "email")
		role := getCSVField(record, colMap, "role")
		eno := getCSVField(record, colMap, "eno")
		filiere := getCSVField(record, colMap, "filiere")
		niveau := getCSVField(record, colMap, "niveau")

		// Validate required fields
		if nom == "" || prenom == "" || email == "" || role == "" {
			results = append(results, ImportResult{
				Nom: nom, Prenom: prenom, Email: email, Role: role,
				Status: "erreur",
				Error:  "Champs obligatoires manquants (nom, prenom, email, role)",
			})
			errorCount++
			continue
		}

		// Validate role
		if role != "etudiant" && role != "enseignant" && role != "admin" {
			results = append(results, ImportResult{
				Nom: nom, Prenom: prenom, Email: email, Role: role,
				Status: "erreur",
				Error:  "Rôle invalide (accepté : etudiant, enseignant, admin)",
			})
			errorCount++
			continue
		}

		// Check if email already exists
		var existing models.User
		if err := config.DB.Where("email = ?", email).First(&existing).Error; err == nil {
			results = append(results, ImportResult{
				Nom: nom, Prenom: prenom, Email: email, Role: role,
				Status: "ignoré",
				Error:  "Email déjà existant",
			})
			errorCount++
			continue
		}

		// Generate and hash temporary password
		tempPassword := generateTempPassword(8)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
		if err != nil {
			results = append(results, ImportResult{
				Nom: nom, Prenom: prenom, Email: email, Role: role,
				Status: "erreur",
				Error:  "Erreur de hashage",
			})
			errorCount++
			continue
		}

		user := models.User{
			Nom:                nom,
			Prenom:             prenom,
			Email:              email,
			Password:           string(hashedPassword),
			Role:               role,
			IsActive:           true,
			MustChangePassword: true,
			ENO:                eno,
			Filiere:            filiere,
			Niveau:             niveau,
		}

		if err := config.DB.Create(&user).Error; err != nil {
			results = append(results, ImportResult{
				Nom: nom, Prenom: prenom, Email: email, Role: role,
				Status: "erreur",
				Error:  "Erreur base de données",
			})
			errorCount++
			continue
		}

		results = append(results, ImportResult{
			Nom:          nom,
			Prenom:       prenom,
			Email:        email,
			Role:         role,
			TempPassword: tempPassword,
			Status:       "créé",
		})
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       fmt.Sprintf("%d utilisateur(s) créé(s), %d erreur(s)", successCount, errorCount),
		"success_count": successCount,
		"error_count":   errorCount,
		"total":         successCount + errorCount,
		"results":       results,
	})
}

// getCSVField safely retrieves a field from a CSV record by column name
func getCSVField(record []string, colMap map[string]int, colName string) string {
	if idx, exists := colMap[colName]; exists && idx < len(record) {
		return strings.TrimSpace(record[idx])
	}
	return ""
}

// generateTempPassword creates a random temporary password
func generateTempPassword(length int) string {
	const charset = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	password := make([]byte, length)
	for i := range password {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		password[i] = charset[n.Int64()]
	}

	// Ensure at least one digit and one uppercase for readability
	digitIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(length)))
	upperIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(length)))


	digits := "23456789"
	uppers := "ABCDEFGHJKLMNPQRSTUVWXYZ"
	dn, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
	un, _ := rand.Int(rand.Reader, big.NewInt(int64(len(uppers))))
	password[digitIdx.Int64()] = digits[dn.Int64()]
	password[upperIdx.Int64()] = uppers[un.Int64()]

	return string(password)
}
