package controllers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/unchk/agent-ia-backend/config"
	"github.com/unchk/agent-ia-backend/models"
)

// SubmitTestPayload is the expected input from the Next.js frontend
type SubmitTestPayload struct {
	EtudiantID uint               `json:"etudiant_id" binding:"required"`
	Reponses   map[uint]string    `json:"reponses"` // map of question_id to answer string
}

// UpdateTestPayload represents the expected request body for updating a test
type UpdateTestPayload struct {
	Titre       string `json:"titre"`
	Description string `json:"description"`
	Domaine     string `json:"domaine"`
	NiveauCible string `json:"niveau_cible"`
}

// SubmitTest handles the submission of a student's test
func SubmitTest(c *gin.Context) {
	testID := c.Param("id")
	var payload SubmitTestPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format de données invalide"})
		return
	}

	// Fetch the questions for this test to calculate the score
	var questions []models.Question
	if err := config.DB.Where("test_niveau_id = ?", testID).Find(&questions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les questions"})
		return
	}

	// Simple scoring logic
	scoresParConcept := make(map[string]int)
	scoreGlobal := 0

	for _, q := range questions {
		if ans, exists := payload.Reponses[q.ID]; exists && ans != "" {
			scoresParConcept[q.Concept] = 80 // Mock score
			scoreGlobal += 80
		} else {
			scoresParConcept[q.Concept] = 30 // Mock low score (lacune)
			scoreGlobal += 30
		}
	}

	scoresDetailJSON, _ := json.Marshal(scoresParConcept)

	resultat := models.Resultat{
		EtudiantID:   payload.EtudiantID,
		TestNiveauID: questions[0].TestNiveauID, 
		ScoreGlobal:  scoreGlobal / len(questions),
		ScoresDetail: string(scoresDetailJSON),
		DatePassage:  time.Now(),
	}

	if err := config.DB.Create(&resultat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Échec de l'enregistrement du résultat"})
		return
	}

	// Send data to Python AI Engine
	aiEngineURL := "http://localhost:8000/api/recommend"
	aiPayload, _ := json.Marshal(map[string]interface{}{
		"resultat_id":   resultat.ID,
		"etudiant_id":   resultat.EtudiantID,
		"student_scores": scoresParConcept,
	})

	resp, err := http.Post(aiEngineURL, "application/json", bytes.NewBuffer(aiPayload))
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusPartialContent, gin.H{
			"message": "Test validé et enregistré. L'agent IA est momentanément indisponible pour générer les recommandations.",
			"resultat": resultat,
		})
		return
	}
	defer resp.Body.Close()

	var aiResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&aiResponse)

	c.JSON(http.StatusOK, gin.H{
		"message": "Test validé avec succès. Recommandations générées.",
		"resultat": resultat,
		"recommendations": aiResponse["recommendations"],
	})
}

// GetTests returns all available tests, optionally with their questions count
func GetTests(c *gin.Context) {
	// Let's create a custom struct to return tests with their questions count
	type TestWithCounts struct {
		models.TestNiveau
		NbQuestions int `json:"nb_questions"`
	}

	var tests []models.TestNiveau
	if err := config.DB.Order("created_at DESC").Find(&tests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des tests"})
		return
	}

	var results []TestWithCounts
	for _, t := range tests {
		var count int64
		config.DB.Model(&models.Question{}).Where("test_niveau_id = ?", t.ID).Count(&count)
		results = append(results, TestWithCounts{
			TestNiveau:  t,
			NbQuestions: int(count),
		})
	}

	c.JSON(http.StatusOK, results)
}

// UpdateTest allows an author (enseignant) to update their test
func UpdateTest(c *gin.Context) {
	id := c.Param("id")
	var test models.TestNiveau

	if err := config.DB.First(&test, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Test non trouvé"})
		return
	}

	// Verify authorization (only auteur or admin)
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")
	if userRole != "admin" && uint(userID.(float64)) != test.AuteurID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vous n'êtes pas autorisé à modifier ce test"})
		return
	}

	var payload UpdateTestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
		return
	}

	if payload.Titre != "" { test.Titre = payload.Titre }
	if payload.Description != "" { test.Description = payload.Description }
	if payload.Domaine != "" { test.Domaine = payload.Domaine }
	if payload.NiveauCible != "" { test.NiveauCible = payload.NiveauCible }

	if err := config.DB.Save(&test).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la mise à jour"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test mis à jour avec succès", "test": test})
}

// DeleteTest allows an author to delete their test
func DeleteTest(c *gin.Context) {
	id := c.Param("id")
	var test models.TestNiveau

	if err := config.DB.First(&test, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Test non trouvé"})
		return
	}

	// Verify authorization
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")
	if userRole != "admin" && uint(userID.(float64)) != test.AuteurID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vous n'êtes pas autorisé à supprimer ce test"})
		return
	}

	// Delete associated questions first (Cascade)
	if err := config.DB.Where("test_niveau_id = ?", test.ID).Delete(&models.Question{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression des questions associées"})
		return
	}

	if err := config.DB.Delete(&test).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression du test"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test et questions associées supprimés avec succès"})
}

// ImportTestsCSV handles bulk import of tests and questions from a CSV file
func ImportTestsCSV(c *gin.Context) {
	userID, _ := c.Get("user_id")
	authorID := uint(userID.(float64))

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fichier CSV requis"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Impossible de lire l'en-tête du CSV"})
		return
	}

	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(strings.ToLower(col))] = i
	}

	requiredCols := []string{"titre_test", "domaine", "question_enonce", "question_concept", "question_points"}
	for _, col := range requiredCols {
		if _, exists := colMap[col]; !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Colonne requise manquante : %s", col)})
			return
		}
	}

	type ImportResult struct {
		TestTitre string `json:"test_titre"`
		Question  string `json:"question"`
		Status    string `json:"status"`
		Error     string `json:"error,omitempty"`
	}

	var results []ImportResult
	testsCache := make(map[string]*models.TestNiveau)
	successCount := 0
	errorCount := 0

	for {
		record, err := reader.Read()
		if err == io.EOF { break }
		if err != nil {
			errorCount++
			continue
		}

		titreTest := getCSVField(record, colMap, "titre_test")
		description := getCSVField(record, colMap, "description_test")
		domaine := getCSVField(record, colMap, "domaine")
		niveau := getCSVField(record, colMap, "niveau_cible")
		qEnonce := getCSVField(record, colMap, "question_enonce")
		qConcept := getCSVField(record, colMap, "question_concept")
		qPointsStr := getCSVField(record, colMap, "question_points")

		if titreTest == "" || qEnonce == "" {
			results = append(results, ImportResult{Status: "erreur", Error: "Titre du test ou énoncé manquant"})
			errorCount++
			continue
		}

		qPoints, _ := strconv.Atoi(qPointsStr)
		if qPoints == 0 { qPoints = 10 } // Default points

		// Get or Create Test
		test, exists := testsCache[titreTest]
		if !exists {
			// Check DB
			var dbTest models.TestNiveau
			if err := config.DB.Where("titre = ? AND auteur_id = ?", titreTest, authorID).First(&dbTest).Error; err != nil {
				// Create new
				dbTest = models.TestNiveau{
					Titre:       titreTest,
					Description: description,
					Domaine:     domaine,
					NiveauCible: niveau,
					AuteurID:    authorID,
				}
				if err := config.DB.Create(&dbTest).Error; err != nil {
					results = append(results, ImportResult{TestTitre: titreTest, Question: qEnonce, Status: "erreur", Error: "Erreur création test"})
					errorCount++
					continue
				}
			}
			test = &dbTest
			testsCache[titreTest] = test
		}

		// Create Question
		question := models.Question{
			TestNiveauID: test.ID,
			Enonce:       qEnonce,
			Concept:      qConcept,
			Points:       qPoints,
		}

		if err := config.DB.Create(&question).Error; err != nil {
			results = append(results, ImportResult{TestTitre: titreTest, Question: qEnonce, Status: "erreur", Error: "Erreur création question"})
			errorCount++
			continue
		}

		results = append(results, ImportResult{TestTitre: titreTest, Question: qEnonce, Status: "créé"})
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       fmt.Sprintf("%d question(s) importée(s) avec succès.", successCount),
		"success_count": successCount,
		"error_count":   errorCount,
		"tests_created": len(testsCache),
		"results":       results,
	})
}
