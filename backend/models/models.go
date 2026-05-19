package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents an actor in the system (Etudiant, Enseignant, Admin)
// Context UNCHK: ENO (Espace Numérique Ouvert), Filiere (e.g., Licence Informatique)
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Nom       string         `json:"nom"`
	Prenom    string         `json:"prenom"`
	Email     string         `gorm:"unique" json:"email"`
	Password  string         `json:"-"` // Not exposed in JSON
	Role               string         `json:"role"` // etudiant, enseignant, admin
	IsActive           bool           `json:"is_active" gorm:"default:true"`
	MustChangePassword bool           `json:"must_change_password" gorm:"default:true"`
	ENO                string         `json:"eno"`  // e.g., ENO de Dakar, ENO de Thiès (UNCHK specific)
	Filiere   string         `json:"filiere"` // e.g., Informatique, Droit
	Niveau    string         `json:"niveau"`  // e.g., L1, L2, L3, M1, M2
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Concept represents a pedagogical concept/skill area (e.g., UML, SQL, POO)
type Concept struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Nom       string    `gorm:"unique" json:"nom"`
	CreatedAt time.Time `json:"created_at"`
}

// TestNiveau represents an evaluation test
type TestNiveau struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Titre       string    `json:"titre"`
	Description string    `json:"description"`
	Domaine     string    `json:"domaine"` // e.g., Programmation, Base de données
	NiveauCible string    `json:"niveau_cible"`
	AuteurID    uint      `json:"auteur_id"` // Reference to User (Enseignant)
	CreatedAt   time.Time `json:"created_at"`
}

// Question represents a question inside a TestNiveau
type Question struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	TestNiveauID uint   `json:"test_niveau_id"`
	Enonce       string `json:"enonce"`
	Concept      string `json:"concept"` // The specific skill/concept evaluated (e.g., "UML", "SQL")
	Points       int    `json:"points"`
}

// Reponse represents a predefined answer for a question
type Reponse struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	QuestionID uint   `json:"question_id"`
	Texte      string `json:"texte"`
	EstCorrect bool   `json:"est_correct"`
}

// Resultat represents the score of a student for a specific test
type Resultat struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	EtudiantID   uint   `json:"etudiant_id"`
	TestNiveauID uint   `json:"test_niveau_id"`
	ScoreGlobal  int    `json:"score_global"`
	ScoresDetail string `json:"scores_detail"` // JSON string storing scores per concept, e.g., {"UML": 40, "SQL": 80}
	DatePassage  time.Time `json:"date_passage"`
}

// RessourcePedagogique represents educational materials
type RessourcePedagogique struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Titre           string    `json:"titre"`
	Type            string    `json:"type"` // e.g., Video, PDF, Exercice
	Lien            string    `json:"lien"`
	Concept         string    `json:"concept"` // Must match Question.Concept
	NiveauDifficulte int      `json:"niveau_difficulte"` // 1 (Débutant), 2 (Intermédiaire), 3 (Avancé)
	Pertinence      int       `json:"pertinence"` // Weight for recommendation algorithm
	AuteurID        uint      `json:"auteur_id"`
	CreatedAt       time.Time `json:"created_at"`
}

// Recommandation connects a student's result to suggested resources
type Recommandation struct {
	ID                     uint      `gorm:"primaryKey" json:"id"`
	ResultatID             uint      `json:"resultat_id"`
	EtudiantID             uint      `json:"etudiant_id"`
	RessourcePedagogiqueID uint      `json:"ressource_pedagogique_id"`
	Ressource              RessourcePedagogique `gorm:"foreignKey:RessourcePedagogiqueID" json:"ressource"`
	Raison                 string    `json:"raison"` // e.g., "Lacune identifiée en UML"
	DateRecommandation     time.Time `json:"date_recommandation"`
}
