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

// QuestionOption is a student-safe option response structure
type QuestionOption struct {
	ID    uint   `json:"id"`
	Texte string `json:"texte"`
}

// QuestionStudentView is a student-safe question structure (does not expose correctness)
type QuestionStudentView struct {
	ID           uint             `json:"id"`
	TestNiveauID uint             `json:"test_niveau_id"`
	Enonce       string           `json:"enonce"`
	Concept      string           `json:"concept"`
	Points       int              `json:"points"`
	Options      []QuestionOption `json:"options"`
}

// UpdateTestPayload represents the expected request body for updating a test
type UpdateTestPayload struct {
	Titre       string `json:"titre"`
	Description string `json:"description"`
	Domaine     string `json:"domaine"`
	NiveauCible string `json:"niveau_cible"`
}

// SubmitTest handles the submission of a student's test with real correction logic
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

	if len(questions) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Test vide ou inexistant"})
		return
	}

	// Dynamic grading logic using DB answers (Reponse)
	pointsParConcept := make(map[string]int)
	pointsGagnesParConcept := make(map[string]int)

	for _, q := range questions {
		pointsParConcept[q.Concept] += q.Points

		var correctRep models.Reponse
		// Find the correct option in DB
		if err := config.DB.Where("question_id = ? AND est_correct = ?", q.ID, true).First(&correctRep).Error; err == nil {
			if ans, exists := payload.Reponses[q.ID]; exists && ans != "" {
				// The answer sent can be the string of the Response ID
				ansID, err := strconv.ParseUint(ans, 10, 32)
				if err == nil && uint(ansID) == correctRep.ID {
					pointsGagnesParConcept[q.Concept] += q.Points
				}
			}
		}
	}

	scoresParConcept := make(map[string]int)
	totalPoints := 0
	totalEarned := 0

	for concept, total := range pointsParConcept {
		earned := pointsGagnesParConcept[concept]
		score := 0
		if total > 0 {
			score = (earned * 100) / total
		}
		scoresParConcept[concept] = score
		totalPoints += total
		totalEarned += earned
	}

	scoreGlobal := 0
	if totalPoints > 0 {
		scoreGlobal = (totalEarned * 100) / totalPoints
	}

	scoresDetailJSON, _ := json.Marshal(scoresParConcept)

	resultat := models.Resultat{
		EtudiantID:   payload.EtudiantID,
		TestNiveauID: questions[0].TestNiveauID, 
		ScoreGlobal:  scoreGlobal,
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

	// Save recommendations to database
	if recs, ok := aiResponse["recommendations"].([]interface{}); ok {
		for _, recInterface := range recs {
			if recMap, ok := recInterface.(map[string]interface{}); ok {
				var ressourceID uint
				if idFloat, ok := recMap["ressource_id"].(float64); ok {
					ressourceID = uint(idFloat)
				}
				
				raison, _ := recMap["raison"].(string)

				if ressourceID > 0 {
					recommendation := models.Recommandation{
						ResultatID:             resultat.ID,
						EtudiantID:             payload.EtudiantID,
						RessourcePedagogiqueID: ressourceID,
						Raison:                 raison,
						DateRecommandation:     time.Now(),
					}
					config.DB.Create(&recommendation)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Test validé avec succès. Recommandations générées et sauvegardées.",
		"resultat": resultat,
		"recommendations": aiResponse["recommendations"],
	})
}

// CreateTestPayload is the expected input for creating a test
type CreateTestPayload struct {
	Titre       string                  `json:"titre" binding:"required"`
	Description string                  `json:"description" binding:"required"`
	Domaine     string                  `json:"domaine" binding:"required"`
	NiveauCible string                  `json:"niveau_cible" binding:"required"`
	Questions   []CreateQuestionPayload `json:"questions" binding:"required"`
}

type CreateQuestionPayload struct {
	Enonce  string `json:"enonce" binding:"required"`
	Concept string `json:"concept" binding:"required"`
	Points  int    `json:"points" binding:"required"`
}

// CreateTest handles the creation of a new evaluation test
func CreateTest(c *gin.Context) {
	var payload CreateTestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides ou incomplètes"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non autorisé"})
		return
	}

	// Create test
	test := models.TestNiveau{
		Titre:       payload.Titre,
		Description: payload.Description,
		Domaine:     payload.Domaine,
		NiveauCible: payload.NiveauCible,
		AuteurID:    userID.(uint),
	}

	if err := config.DB.Create(&test).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du test"})
		return
	}

	// Create questions
	for _, qPayload := range payload.Questions {
		question := models.Question{
			TestNiveauID: test.ID,
			Enonce:       qPayload.Enonce,
			Concept:      qPayload.Concept,
			Points:       qPayload.Points,
		}
		config.DB.Create(&question)
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Test créé avec succès", "test": test})
}

// GetTests returns all available tests, optionally with their questions count and concepts
func GetTests(c *gin.Context) {
	// Custom struct to return tests with their questions count and covered concepts
	type TestWithCounts struct {
		models.TestNiveau
		NbQuestions int      `json:"nb_questions"`
		Concepts    []string `json:"concepts"`
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

		// Fetch distinct concepts covered by this test's questions
		var testConcepts []string
		config.DB.Model(&models.Question{}).Where("test_niveau_id = ?", t.ID).Distinct("concept").Pluck("concept", &testConcepts)

		results = append(results, TestWithCounts{
			TestNiveau:  t,
			NbQuestions: int(count),
			Concepts:    testConcepts,
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
	if userRole != "admin" && userID.(uint) != test.AuteurID {
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
	if userRole != "admin" && userID.(uint) != test.AuteurID {
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
	authorID := userID.(uint)

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

// GetRecommendations returns the recommendations for the logged-in student
func GetRecommendations(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non autorisé"})
		return
	}
	studentID := userID.(uint)

	var recommendations []models.Recommandation
	if err := config.DB.Where("etudiant_id = ?", studentID).Preload("Ressource").Order("date_recommandation DESC").Find(&recommendations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les recommandations"})
		return
	}

	c.JSON(http.StatusOK, recommendations)
}

// GetConcepts returns a list of unique concepts from the questions table
func GetConcepts(c *gin.Context) {
	var concepts []string
	if err := config.DB.Model(&models.Question{}).Distinct("concept").Pluck("concept", &concepts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les concepts"})
		return
	}
	c.JSON(http.StatusOK, concepts)
}

// GetTestByID returns details of a test along with its questions (safe for students)
func GetTestByID(c *gin.Context) {
	testID := c.Param("id")
	var test models.TestNiveau

	if err := config.DB.First(&test, testID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Test non trouvé"})
		return
	}

	var questions []models.Question
	if err := config.DB.Where("test_niveau_id = ?", test.ID).Find(&questions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les questions"})
		return
	}

	var studentQuestions []QuestionStudentView
	for _, q := range questions {
		var responses []models.Reponse
		config.DB.Where("question_id = ?", q.ID).Find(&responses)

		var options []QuestionOption
		for _, r := range responses {
			options = append(options, QuestionOption{
				ID:    r.ID,
				Texte: r.Texte,
			})
		}

		studentQuestions = append(studentQuestions, QuestionStudentView{
			ID:           q.ID,
			TestNiveauID: q.TestNiveauID,
			Enonce:       q.Enonce,
			Concept:      q.Concept,
			Points:       q.Points,
			Options:      options,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"test":      test,
		"questions": studentQuestions,
	})
}

// GetStudentResults returns the test results (scores) of the logged-in student
func GetStudentResults(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non autorisé"})
		return
	}
	studentID := userID.(uint)

	// Custom struct to return result with test details
	type ResultWithTest struct {
		models.Resultat
		TestTitre   string `json:"test_titre"`
		TestDomaine string `json:"test_domaine"`
	}

	var dbResults []models.Resultat
	if err := config.DB.Where("etudiant_id = ?", studentID).Order("date_passage DESC").Find(&dbResults).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les résultats"})
		return
	}

	results := []ResultWithTest{}
	for _, r := range dbResults {
		var test models.TestNiveau
		config.DB.Select("titre", "domaine").First(&test, r.TestNiveauID)
		results = append(results, ResultWithTest{
			Resultat:    r,
			TestTitre:   test.Titre,
			TestDomaine: test.Domaine,
		})
	}

	c.JSON(http.StatusOK, results)
}
