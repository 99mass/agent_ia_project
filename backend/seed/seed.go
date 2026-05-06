package seed

import (
	"log"
	"strings"

	"github.com/unchk/agent-ia-backend/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// seedUser creates a user if not exists, or updates the password if it's not a valid bcrypt hash
func seedUser(db *gorm.DB, user models.User) models.User {
	var existing models.User
	result := db.Where("email = ?", user.Email).First(&existing)

	if result.Error != nil {
		// User doesn't exist, create it
		db.Create(&user)
		return user
	}

	// User exists — check if password needs updating (not a bcrypt hash)
	if !strings.HasPrefix(existing.Password, "$2a$") && !strings.HasPrefix(existing.Password, "$2b$") {
		existing.Password = user.Password
		existing.IsActive = user.IsActive
		existing.MustChangePassword = user.MustChangePassword
		existing.Role = user.Role
		db.Save(&existing)
		log.Printf("Mot de passe mis à jour pour: %s", existing.Email)
	}

	return existing
}

func SeedDatabase(db *gorm.DB) {
	// Hash passwords using bcrypt
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	enseignantHash, _ := bcrypt.GenerateFromPassword([]byte("enseignant123"), bcrypt.DefaultCost)
	etudiantHash, _ := bcrypt.GenerateFromPassword([]byte("etudiant123"), bcrypt.DefaultCost)

	// Seed Admin user
	_ = seedUser(db, models.User{
		Nom:                "Ndiaye",
		Prenom:             "Moussa",
		Email:              "admin@unchk.sn",
		Password:           string(adminHash),
		Role:               "admin",
		IsActive:           true,
		MustChangePassword: false,
		ENO:                "Dakar",
	})

	// Seed Enseignant
	enseignant := seedUser(db, models.User{
		Nom:                "Diop",
		Prenom:             "Amadou",
		Email:              "amadou.diop@unchk.sn",
		Password:           string(enseignantHash),
		Role:               "enseignant",
		IsActive:           true,
		MustChangePassword: false,
		ENO:                "Dakar",
		Filiere:            "Génie Logiciel",
		Niveau:             "Master",
	})

	// Seed Étudiant
	seedUser(db, models.User{
		Nom:                "Sarr",
		Prenom:             "Fatou",
		Email:              "fatou.sarr@unchk.edu.sn",
		Password:           string(etudiantHash),
		Role:               "etudiant",
		IsActive:           true,
		MustChangePassword: false,
		ENO:                "Thiès",
		Filiere:            "Génie Logiciel",
		Niveau:             "Master 2",
	})

	// Seed Test de Niveau
	test1 := models.TestNiveau{
		Titre:       "Évaluation des prérequis en Génie Logiciel",
		Description: "Test d'entrée pour le cours d'Architecture des Systèmes de l'UNCHK.",
		Domaine:     "Génie Logiciel",
		NiveauCible: "Master 2",
		AuteurID:    enseignant.ID,
	}
	db.FirstOrCreate(&test1, models.TestNiveau{Titre: test1.Titre})

	// Seed Questions
	questions := []models.Question{
		{TestNiveauID: test1.ID, Enonce: "Que signifie UML ?", Concept: "UML", Points: 10},
		{TestNiveauID: test1.ID, Enonce: "Quelle est la différence entre une classe et un objet ?", Concept: "POO", Points: 10},
		{TestNiveauID: test1.ID, Enonce: "Quel modèle de cycle de vie est basé sur des itérations (Sprints) ?", Concept: "Agile", Points: 10},
	}
	for _, q := range questions {
		db.FirstOrCreate(&q, models.Question{Enonce: q.Enonce})
	}

	// Seed Ressources Pédagogiques
	ressources := []models.RessourcePedagogique{
		{
			Titre:            "Introduction aux diagrammes de classes UML",
			Type:             "Vidéo",
			Lien:             "https://lms.unchk.sn/ressources/uml_intro.mp4",
			Concept:          "UML",
			NiveauDifficulte: 1,
			Pertinence:       5,
			AuteurID:         enseignant.ID,
		},
		{
			Titre:            "Les concepts fondamentaux de la Programmation Orientée Objet",
			Type:             "PDF",
			Lien:             "https://lms.unchk.sn/ressources/poo_fondamentaux.pdf",
			Concept:          "POO",
			NiveauDifficulte: 1,
			Pertinence:       4,
			AuteurID:         enseignant.ID,
		},
		{
			Titre:            "Guide pratique Scrum et Agilité",
			Type:             "Exercice interactif",
			Lien:             "https://lms.unchk.sn/ressources/agile_guide.html",
			Concept:          "Agile",
			NiveauDifficulte: 2,
			Pertinence:       5,
			AuteurID:         enseignant.ID,
		},
	}
	for _, r := range ressources {
		db.FirstOrCreate(&r, models.RessourcePedagogique{Titre: r.Titre})
	}

	log.Println("Données contextuelles UNCHK injectées avec succès.")
}

// Ensure admin user exists (for migration from old seeds)
func init() {
	// This is handled by seedUser function which checks for non-bcrypt passwords
}
