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

// seedTest creates a test if not exists by title, returns the test
func seedTest(db *gorm.DB, test models.TestNiveau) models.TestNiveau {
	db.FirstOrCreate(&test, models.TestNiveau{Titre: test.Titre})
	return test
}

// seedQuestion creates a question if not exists by enonce, returns the question
func seedQuestion(db *gorm.DB, q models.Question) models.Question {
	db.FirstOrCreate(&q, models.Question{Enonce: q.Enonce})
	return q
}

// seedReponses creates responses for a question (found by enonce)
func seedReponses(db *gorm.DB, enonce string, reponses []models.Reponse) {
	var q models.Question
	if err := db.Where("enonce = ?", enonce).First(&q).Error; err == nil {
		for _, r := range reponses {
			r.QuestionID = q.ID
			db.FirstOrCreate(&r, models.Reponse{QuestionID: r.QuestionID, Texte: r.Texte})
		}
	}
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

	// Seed Concepts
	defaultConcepts := []string{"UML", "POO", "Agile", "SQL", "Algorithmique", "Réseaux", "Base de données", "HTML/CSS", "Programmation", "Architecture logicielle"}
	for _, nom := range defaultConcepts {
		concept := models.Concept{Nom: nom}
		db.FirstOrCreate(&concept, models.Concept{Nom: nom})
	}

	authorID := enseignant.ID

	// ──────────────────────────────────────────────
	// TEST 1: UML — Évaluation des prérequis en Génie Logiciel (existing)
	// ──────────────────────────────────────────────
	test1 := seedTest(db, models.TestNiveau{
		Titre:       "Évaluation des prérequis en Génie Logiciel",
		Description: "Test d'entrée pour le cours d'Architecture des Systèmes de l'UNCHK.",
		Domaine:     "Génie Logiciel",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test1.ID, Enonce: "Que signifie UML ?", Concept: "UML", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test1.ID, Enonce: "Quelle est la différence entre une classe et un objet ?", Concept: "POO", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test1.ID, Enonce: "Quel modèle de cycle de vie est basé sur des itérations (Sprints) ?", Concept: "Agile", Points: 10})

	// Reponses for test1
	seedReponses(db, "Que signifie UML ?", []models.Reponse{
		{Texte: "Unified Modeling Language — un langage de modélisation standardisé pour visualiser la conception d'un système.", EstCorrect: true},
		{Texte: "Universal Machine Learning — un framework d'apprentissage automatique.", EstCorrect: false},
		{Texte: "User Management Layer — une couche de gestion des utilisateurs.", EstCorrect: false},
		{Texte: "Unified Memory Loader — un outil de gestion de mémoire.", EstCorrect: false},
	})
	seedReponses(db, "Quelle est la différence entre une classe et un objet ?", []models.Reponse{
		{Texte: "Il n'y a aucune différence, ce sont des synonymes.", EstCorrect: false},
		{Texte: "Une classe est un modèle (blueprint), un objet est une instance concrète de cette classe.", EstCorrect: true},
		{Texte: "Un objet est un modèle, une classe est une instance.", EstCorrect: false},
		{Texte: "Une classe ne contient que des méthodes, un objet ne contient que des attributs.", EstCorrect: false},
	})
	seedReponses(db, "Quel modèle de cycle de vie est basé sur des itérations (Sprints) ?", []models.Reponse{
		{Texte: "Le modèle en cascade (Waterfall).", EstCorrect: false},
		{Texte: "Le modèle en V.", EstCorrect: false},
		{Texte: "La méthode Scrum (Agile).", EstCorrect: true},
		{Texte: "Le modèle en spirale.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 2: UML — Diagrammes UML avancés
	// ──────────────────────────────────────────────
	test2 := seedTest(db, models.TestNiveau{
		Titre:       "Diagrammes UML avancés",
		Description: "Évaluation approfondie sur les diagrammes de séquence, d'activité et de cas d'utilisation.",
		Domaine:     "Génie Logiciel",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test2.ID, Enonce: "Quel diagramme UML représente l'interaction entre les objets dans le temps ?", Concept: "UML", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test2.ID, Enonce: "Dans un diagramme de cas d'utilisation, que représente un acteur ?", Concept: "UML", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test2.ID, Enonce: "Quel est le rôle d'un diagramme d'activité ?", Concept: "UML", Points: 10})

	seedReponses(db, "Quel diagramme UML représente l'interaction entre les objets dans le temps ?", []models.Reponse{
		{Texte: "Diagramme de classes.", EstCorrect: false},
		{Texte: "Diagramme de séquence.", EstCorrect: true},
		{Texte: "Diagramme de déploiement.", EstCorrect: false},
		{Texte: "Diagramme de composants.", EstCorrect: false},
	})
	seedReponses(db, "Dans un diagramme de cas d'utilisation, que représente un acteur ?", []models.Reponse{
		{Texte: "Un composant interne du système.", EstCorrect: false},
		{Texte: "Une entité externe qui interagit avec le système.", EstCorrect: true},
		{Texte: "Une base de données.", EstCorrect: false},
		{Texte: "Un processus métier automatisé.", EstCorrect: false},
	})
	seedReponses(db, "Quel est le rôle d'un diagramme d'activité ?", []models.Reponse{
		{Texte: "Modéliser la structure statique des classes.", EstCorrect: false},
		{Texte: "Représenter le flux de contrôle d'un processus ou algorithme.", EstCorrect: true},
		{Texte: "Décrire l'architecture physique du déploiement.", EstCorrect: false},
		{Texte: "Montrer les relations d'héritage uniquement.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 3: POO — Fondamentaux de la POO
	// ──────────────────────────────────────────────
	test3 := seedTest(db, models.TestNiveau{
		Titre:       "Fondamentaux de la Programmation Orientée Objet",
		Description: "Évaluation des concepts de base de la POO : encapsulation, héritage, polymorphisme.",
		Domaine:     "Programmation",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test3.ID, Enonce: "Quel pilier de la POO permet de masquer les détails d'implémentation ?", Concept: "POO", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test3.ID, Enonce: "Qu'est-ce que le polymorphisme en POO ?", Concept: "POO", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test3.ID, Enonce: "Quelle est la relation 'est-un' en POO ?", Concept: "POO", Points: 10})

	seedReponses(db, "Quel pilier de la POO permet de masquer les détails d'implémentation ?", []models.Reponse{
		{Texte: "Le polymorphisme.", EstCorrect: false},
		{Texte: "L'encapsulation.", EstCorrect: true},
		{Texte: "L'héritage.", EstCorrect: false},
		{Texte: "L'abstraction.", EstCorrect: false},
	})
	seedReponses(db, "Qu'est-ce que le polymorphisme en POO ?", []models.Reponse{
		{Texte: "La capacité d'un objet à prendre plusieurs formes et à répondre différemment au même message.", EstCorrect: true},
		{Texte: "La capacité de créer des copies d'un objet.", EstCorrect: false},
		{Texte: "Le fait de cacher les attributs privés.", EstCorrect: false},
		{Texte: "Un synonyme de l'héritage multiple.", EstCorrect: false},
	})
	seedReponses(db, "Quelle est la relation 'est-un' en POO ?", []models.Reponse{
		{Texte: "La composition.", EstCorrect: false},
		{Texte: "L'agrégation.", EstCorrect: false},
		{Texte: "L'héritage.", EstCorrect: true},
		{Texte: "L'association.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 4: POO — Design Patterns
	// ──────────────────────────────────────────────
	test4 := seedTest(db, models.TestNiveau{
		Titre:       "Design Patterns et POO avancée",
		Description: "Évaluation sur les patrons de conception : Singleton, Observer, Factory et MVC.",
		Domaine:     "Programmation",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test4.ID, Enonce: "Quel pattern garantit qu'une seule instance d'une classe existe ?", Concept: "POO", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test4.ID, Enonce: "Dans le pattern MVC, quel composant gère la logique métier ?", Concept: "POO", Points: 10})

	seedReponses(db, "Quel pattern garantit qu'une seule instance d'une classe existe ?", []models.Reponse{
		{Texte: "Observer.", EstCorrect: false},
		{Texte: "Factory.", EstCorrect: false},
		{Texte: "Singleton.", EstCorrect: true},
		{Texte: "Strategy.", EstCorrect: false},
	})
	seedReponses(db, "Dans le pattern MVC, quel composant gère la logique métier ?", []models.Reponse{
		{Texte: "La Vue (View).", EstCorrect: false},
		{Texte: "Le Modèle (Model).", EstCorrect: true},
		{Texte: "Le Contrôleur (Controller).", EstCorrect: false},
		{Texte: "Le Routeur (Router).", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 5: Agile — Méthodologies Agiles
	// ──────────────────────────────────────────────
	test5 := seedTest(db, models.TestNiveau{
		Titre:       "Méthodologies Agiles et Gestion de Projet",
		Description: "Évaluation sur Scrum, Kanban, les rôles Agile et les cérémonies.",
		Domaine:     "Gestion de Projet",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test5.ID, Enonce: "Quel est le rôle du Product Owner dans Scrum ?", Concept: "Agile", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test5.ID, Enonce: "Qu'est-ce qu'un Sprint Review ?", Concept: "Agile", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test5.ID, Enonce: "Quelle est la durée recommandée d'un Daily Standup ?", Concept: "Agile", Points: 10})

	seedReponses(db, "Quel est le rôle du Product Owner dans Scrum ?", []models.Reponse{
		{Texte: "Écrire le code source de l'application.", EstCorrect: false},
		{Texte: "Définir et prioriser le Product Backlog en représentant les besoins métier.", EstCorrect: true},
		{Texte: "Gérer l'infrastructure serveur.", EstCorrect: false},
		{Texte: "Animer les Daily Standups.", EstCorrect: false},
	})
	seedReponses(db, "Qu'est-ce qu'un Sprint Review ?", []models.Reponse{
		{Texte: "Une réunion quotidienne de 15 minutes.", EstCorrect: false},
		{Texte: "Une démonstration du travail accompli durant le Sprint aux parties prenantes.", EstCorrect: true},
		{Texte: "Une session de planification du prochain Sprint.", EstCorrect: false},
		{Texte: "Un audit de sécurité du code.", EstCorrect: false},
	})
	seedReponses(db, "Quelle est la durée recommandée d'un Daily Standup ?", []models.Reponse{
		{Texte: "1 heure.", EstCorrect: false},
		{Texte: "30 minutes.", EstCorrect: false},
		{Texte: "15 minutes maximum.", EstCorrect: true},
		{Texte: "2 heures.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 6: SQL — Fondamentaux SQL
	// ──────────────────────────────────────────────
	test6 := seedTest(db, models.TestNiveau{
		Titre:       "Fondamentaux du langage SQL",
		Description: "Évaluation sur les requêtes SELECT, INSERT, UPDATE, DELETE et les jointures.",
		Domaine:     "Base de données",
		NiveauCible: "Licence 3",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test6.ID, Enonce: "Quelle commande SQL permet de récupérer des données d'une table ?", Concept: "SQL", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test6.ID, Enonce: "Quelle clause SQL permet de filtrer les résultats d'une requête ?", Concept: "SQL", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test6.ID, Enonce: "Quel type de jointure retourne toutes les lignes des deux tables ?", Concept: "SQL", Points: 10})

	seedReponses(db, "Quelle commande SQL permet de récupérer des données d'une table ?", []models.Reponse{
		{Texte: "INSERT.", EstCorrect: false},
		{Texte: "SELECT.", EstCorrect: true},
		{Texte: "UPDATE.", EstCorrect: false},
		{Texte: "DELETE.", EstCorrect: false},
	})
	seedReponses(db, "Quelle clause SQL permet de filtrer les résultats d'une requête ?", []models.Reponse{
		{Texte: "ORDER BY.", EstCorrect: false},
		{Texte: "GROUP BY.", EstCorrect: false},
		{Texte: "WHERE.", EstCorrect: true},
		{Texte: "HAVING.", EstCorrect: false},
	})
	seedReponses(db, "Quel type de jointure retourne toutes les lignes des deux tables ?", []models.Reponse{
		{Texte: "INNER JOIN.", EstCorrect: false},
		{Texte: "LEFT JOIN.", EstCorrect: false},
		{Texte: "FULL OUTER JOIN.", EstCorrect: true},
		{Texte: "CROSS JOIN.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 7: SQL — SQL avancé
	// ──────────────────────────────────────────────
	test7 := seedTest(db, models.TestNiveau{
		Titre:       "SQL avancé : sous-requêtes et optimisation",
		Description: "Évaluation sur les sous-requêtes, les index, les vues et les transactions.",
		Domaine:     "Base de données",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test7.ID, Enonce: "Qu'est-ce qu'une sous-requête (subquery) en SQL ?", Concept: "SQL", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test7.ID, Enonce: "À quoi sert un index dans une base de données ?", Concept: "SQL", Points: 10})

	seedReponses(db, "Qu'est-ce qu'une sous-requête (subquery) en SQL ?", []models.Reponse{
		{Texte: "Une requête SQL imbriquée à l'intérieur d'une autre requête.", EstCorrect: true},
		{Texte: "Une requête qui ne retourne aucun résultat.", EstCorrect: false},
		{Texte: "Un synonyme de procédure stockée.", EstCorrect: false},
		{Texte: "Une requête exécutée en parallèle.", EstCorrect: false},
	})
	seedReponses(db, "À quoi sert un index dans une base de données ?", []models.Reponse{
		{Texte: "À chiffrer les données sensibles.", EstCorrect: false},
		{Texte: "À accélérer la recherche et la récupération des données.", EstCorrect: true},
		{Texte: "À sauvegarder automatiquement les données.", EstCorrect: false},
		{Texte: "À limiter l'accès aux utilisateurs.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 8: Algorithmique
	// ──────────────────────────────────────────────
	test8 := seedTest(db, models.TestNiveau{
		Titre:       "Algorithmique et structures de données",
		Description: "Évaluation sur les algorithmes de tri, de recherche et la complexité.",
		Domaine:     "Informatique fondamentale",
		NiveauCible: "Licence 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test8.ID, Enonce: "Quelle est la complexité temporelle du tri par insertion dans le pire cas ?", Concept: "Algorithmique", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test8.ID, Enonce: "Quelle structure de données fonctionne selon le principe FIFO ?", Concept: "Algorithmique", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test8.ID, Enonce: "Quel algorithme de recherche nécessite un tableau trié ?", Concept: "Algorithmique", Points: 10})

	seedReponses(db, "Quelle est la complexité temporelle du tri par insertion dans le pire cas ?", []models.Reponse{
		{Texte: "O(n).", EstCorrect: false},
		{Texte: "O(n log n).", EstCorrect: false},
		{Texte: "O(n²).", EstCorrect: true},
		{Texte: "O(log n).", EstCorrect: false},
	})
	seedReponses(db, "Quelle structure de données fonctionne selon le principe FIFO ?", []models.Reponse{
		{Texte: "La pile (Stack).", EstCorrect: false},
		{Texte: "La file (Queue).", EstCorrect: true},
		{Texte: "L'arbre binaire.", EstCorrect: false},
		{Texte: "La table de hachage.", EstCorrect: false},
	})
	seedReponses(db, "Quel algorithme de recherche nécessite un tableau trié ?", []models.Reponse{
		{Texte: "Recherche linéaire.", EstCorrect: false},
		{Texte: "Recherche dichotomique (binaire).", EstCorrect: true},
		{Texte: "Recherche par hachage.", EstCorrect: false},
		{Texte: "Recherche en largeur (BFS).", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 9: Algorithmique — Récursivité et graphes
	// ──────────────────────────────────────────────
	test9 := seedTest(db, models.TestNiveau{
		Titre:       "Récursivité et algorithmes sur les graphes",
		Description: "Évaluation sur la récursivité, les parcours de graphes BFS/DFS et Dijkstra.",
		Domaine:     "Informatique fondamentale",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test9.ID, Enonce: "Qu'est-ce qu'une condition d'arrêt dans une fonction récursive ?", Concept: "Algorithmique", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test9.ID, Enonce: "Quel algorithme trouve le plus court chemin dans un graphe pondéré ?", Concept: "Algorithmique", Points: 10})

	seedReponses(db, "Qu'est-ce qu'une condition d'arrêt dans une fonction récursive ?", []models.Reponse{
		{Texte: "Un appel récursif supplémentaire.", EstCorrect: false},
		{Texte: "Le cas de base qui empêche les appels infinis et termine la récursion.", EstCorrect: true},
		{Texte: "Une boucle for imbriquée.", EstCorrect: false},
		{Texte: "Un mécanisme de parallélisme.", EstCorrect: false},
	})
	seedReponses(db, "Quel algorithme trouve le plus court chemin dans un graphe pondéré ?", []models.Reponse{
		{Texte: "Le tri à bulles.", EstCorrect: false},
		{Texte: "L'algorithme de Dijkstra.", EstCorrect: true},
		{Texte: "Le parcours en profondeur (DFS).", EstCorrect: false},
		{Texte: "L'algorithme de Kruskal.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 10: Réseaux — Fondamentaux des réseaux
	// ──────────────────────────────────────────────
	test10 := seedTest(db, models.TestNiveau{
		Titre:       "Fondamentaux des réseaux informatiques",
		Description: "Évaluation sur le modèle OSI, TCP/IP, les protocoles réseau et l'adressage IP.",
		Domaine:     "Réseaux",
		NiveauCible: "Licence 3",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test10.ID, Enonce: "Combien de couches comporte le modèle OSI ?", Concept: "Réseaux", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test10.ID, Enonce: "Quel protocole assure un transport fiable des données ?", Concept: "Réseaux", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test10.ID, Enonce: "Quelle est la couche du modèle OSI responsable du routage ?", Concept: "Réseaux", Points: 10})

	seedReponses(db, "Combien de couches comporte le modèle OSI ?", []models.Reponse{
		{Texte: "4 couches.", EstCorrect: false},
		{Texte: "5 couches.", EstCorrect: false},
		{Texte: "7 couches.", EstCorrect: true},
		{Texte: "3 couches.", EstCorrect: false},
	})
	seedReponses(db, "Quel protocole assure un transport fiable des données ?", []models.Reponse{
		{Texte: "UDP.", EstCorrect: false},
		{Texte: "TCP.", EstCorrect: true},
		{Texte: "ICMP.", EstCorrect: false},
		{Texte: "ARP.", EstCorrect: false},
	})
	seedReponses(db, "Quelle est la couche du modèle OSI responsable du routage ?", []models.Reponse{
		{Texte: "Couche Transport.", EstCorrect: false},
		{Texte: "Couche Liaison de données.", EstCorrect: false},
		{Texte: "Couche Réseau.", EstCorrect: true},
		{Texte: "Couche Session.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 11: Réseaux — Sécurité réseau
	// ──────────────────────────────────────────────
	test11 := seedTest(db, models.TestNiveau{
		Titre:       "Sécurité des réseaux et protocoles",
		Description: "Évaluation sur les pare-feux, VPN, chiffrement et menaces réseau.",
		Domaine:     "Réseaux",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test11.ID, Enonce: "Quel est le rôle principal d'un pare-feu (firewall) ?", Concept: "Réseaux", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test11.ID, Enonce: "Quel protocole sécurise les communications HTTPS ?", Concept: "Réseaux", Points: 10})

	seedReponses(db, "Quel est le rôle principal d'un pare-feu (firewall) ?", []models.Reponse{
		{Texte: "Accélérer la connexion Internet.", EstCorrect: false},
		{Texte: "Filtrer le trafic réseau entrant et sortant selon des règles de sécurité.", EstCorrect: true},
		{Texte: "Stocker les fichiers en réseau.", EstCorrect: false},
		{Texte: "Compresser les paquets réseau.", EstCorrect: false},
	})
	seedReponses(db, "Quel protocole sécurise les communications HTTPS ?", []models.Reponse{
		{Texte: "FTP.", EstCorrect: false},
		{Texte: "TLS/SSL.", EstCorrect: true},
		{Texte: "SMTP.", EstCorrect: false},
		{Texte: "DHCP.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 12: Base de données — Conception BDD
	// ──────────────────────────────────────────────
	test12 := seedTest(db, models.TestNiveau{
		Titre:       "Conception et modélisation des bases de données",
		Description: "Évaluation sur le modèle E/R, la normalisation et les contraintes d'intégrité.",
		Domaine:     "Base de données",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test12.ID, Enonce: "Qu'est-ce qu'une clé primaire dans une base de données relationnelle ?", Concept: "Base de données", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test12.ID, Enonce: "Quel est l'objectif de la normalisation des bases de données ?", Concept: "Base de données", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test12.ID, Enonce: "Qu'est-ce qu'une clé étrangère ?", Concept: "Base de données", Points: 10})

	seedReponses(db, "Qu'est-ce qu'une clé primaire dans une base de données relationnelle ?", []models.Reponse{
		{Texte: "Un attribut qui identifie de manière unique chaque enregistrement d'une table.", EstCorrect: true},
		{Texte: "Un attribut qui peut contenir des valeurs nulles.", EstCorrect: false},
		{Texte: "Un attribut qui relie deux tables entre elles.", EstCorrect: false},
		{Texte: "Un index automatique créé par le SGBD.", EstCorrect: false},
	})
	seedReponses(db, "Quel est l'objectif de la normalisation des bases de données ?", []models.Reponse{
		{Texte: "Augmenter la taille de la base de données.", EstCorrect: false},
		{Texte: "Réduire la redondance des données et éviter les anomalies de mise à jour.", EstCorrect: true},
		{Texte: "Accélérer toutes les requêtes.", EstCorrect: false},
		{Texte: "Supprimer les index.", EstCorrect: false},
	})
	seedReponses(db, "Qu'est-ce qu'une clé étrangère ?", []models.Reponse{
		{Texte: "Un identifiant unique dans la table courante.", EstCorrect: false},
		{Texte: "Un attribut qui référence la clé primaire d'une autre table.", EstCorrect: true},
		{Texte: "Un type de données spécial pour les chaînes de caractères.", EstCorrect: false},
		{Texte: "Un synonyme de clé primaire composite.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 13: HTML/CSS — Développement Web Frontend
	// ──────────────────────────────────────────────
	test13 := seedTest(db, models.TestNiveau{
		Titre:       "Fondamentaux HTML et CSS",
		Description: "Évaluation sur la structure HTML5, le CSS3, Flexbox et le responsive design.",
		Domaine:     "Développement Web",
		NiveauCible: "Licence 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test13.ID, Enonce: "Quelle balise HTML5 définit le contenu principal d'une page ?", Concept: "HTML/CSS", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test13.ID, Enonce: "Quelle propriété CSS permet de créer un layout flexible ?", Concept: "HTML/CSS", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test13.ID, Enonce: "À quoi sert une media query en CSS ?", Concept: "HTML/CSS", Points: 10})

	seedReponses(db, "Quelle balise HTML5 définit le contenu principal d'une page ?", []models.Reponse{
		{Texte: "<div>.", EstCorrect: false},
		{Texte: "<main>.", EstCorrect: true},
		{Texte: "<section>.", EstCorrect: false},
		{Texte: "<body>.", EstCorrect: false},
	})
	seedReponses(db, "Quelle propriété CSS permet de créer un layout flexible ?", []models.Reponse{
		{Texte: "display: block.", EstCorrect: false},
		{Texte: "display: flex.", EstCorrect: true},
		{Texte: "display: inline.", EstCorrect: false},
		{Texte: "display: table.", EstCorrect: false},
	})
	seedReponses(db, "À quoi sert une media query en CSS ?", []models.Reponse{
		{Texte: "À importer des polices externes.", EstCorrect: false},
		{Texte: "À adapter le style en fonction de la taille de l'écran ou du dispositif.", EstCorrect: true},
		{Texte: "À animer les éléments HTML.", EstCorrect: false},
		{Texte: "À gérer les événements JavaScript.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 14: HTML/CSS — CSS avancé
	// ──────────────────────────────────────────────
	test14 := seedTest(db, models.TestNiveau{
		Titre:       "CSS avancé : Grid, animations et variables",
		Description: "Évaluation sur CSS Grid, les animations CSS, les variables custom et les pseudo-classes.",
		Domaine:     "Développement Web",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test14.ID, Enonce: "Quelle propriété CSS permet de définir une grille à 3 colonnes ?", Concept: "HTML/CSS", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test14.ID, Enonce: "Comment déclare-t-on une variable CSS custom ?", Concept: "HTML/CSS", Points: 10})

	seedReponses(db, "Quelle propriété CSS permet de définir une grille à 3 colonnes ?", []models.Reponse{
		{Texte: "grid-template-columns: repeat(3, 1fr).", EstCorrect: true},
		{Texte: "flex-direction: column.", EstCorrect: false},
		{Texte: "columns: 3.", EstCorrect: false},
		{Texte: "display: grid-3.", EstCorrect: false},
	})
	seedReponses(db, "Comment déclare-t-on une variable CSS custom ?", []models.Reponse{
		{Texte: "$variable: value.", EstCorrect: false},
		{Texte: "--nom-variable: value.", EstCorrect: true},
		{Texte: "@variable: value.", EstCorrect: false},
		{Texte: "var(nom): value.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 15: Programmation — Fondamentaux
	// ──────────────────────────────────────────────
	test15 := seedTest(db, models.TestNiveau{
		Titre:       "Fondamentaux de la programmation",
		Description: "Évaluation sur les variables, boucles, conditions, fonctions et types de données.",
		Domaine:     "Programmation",
		NiveauCible: "Licence 1",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test15.ID, Enonce: "Quelle structure de contrôle permet de répéter un bloc d'instructions ?", Concept: "Programmation", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test15.ID, Enonce: "Qu'est-ce qu'une fonction en programmation ?", Concept: "Programmation", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test15.ID, Enonce: "Quelle est la différence entre un paramètre et un argument ?", Concept: "Programmation", Points: 10})

	seedReponses(db, "Quelle structure de contrôle permet de répéter un bloc d'instructions ?", []models.Reponse{
		{Texte: "Une condition if/else.", EstCorrect: false},
		{Texte: "Une boucle (for, while).", EstCorrect: true},
		{Texte: "Un switch/case.", EstCorrect: false},
		{Texte: "Une déclaration de variable.", EstCorrect: false},
	})
	seedReponses(db, "Qu'est-ce qu'une fonction en programmation ?", []models.Reponse{
		{Texte: "Un bloc de code réutilisable qui effectue une tâche spécifique.", EstCorrect: true},
		{Texte: "Un type de variable spécial.", EstCorrect: false},
		{Texte: "Un fichier de configuration.", EstCorrect: false},
		{Texte: "Un outil de débogage.", EstCorrect: false},
	})
	seedReponses(db, "Quelle est la différence entre un paramètre et un argument ?", []models.Reponse{
		{Texte: "Il n'y a aucune différence.", EstCorrect: false},
		{Texte: "Un paramètre est dans la définition de la fonction, un argument est la valeur passée lors de l'appel.", EstCorrect: true},
		{Texte: "Un argument est dans la définition, un paramètre est passé à l'appel.", EstCorrect: false},
		{Texte: "Un paramètre est toujours un entier.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 16: Programmation — Programmation système
	// ──────────────────────────────────────────────
	test16 := seedTest(db, models.TestNiveau{
		Titre:       "Programmation système et gestion de mémoire",
		Description: "Évaluation sur les pointeurs, l'allocation mémoire, les processus et les threads.",
		Domaine:     "Programmation",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test16.ID, Enonce: "Qu'est-ce qu'un pointeur en programmation ?", Concept: "Programmation", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test16.ID, Enonce: "Quelle est la différence entre un processus et un thread ?", Concept: "Programmation", Points: 10})

	seedReponses(db, "Qu'est-ce qu'un pointeur en programmation ?", []models.Reponse{
		{Texte: "Une variable qui stocke l'adresse mémoire d'une autre variable.", EstCorrect: true},
		{Texte: "Un type de boucle spécial.", EstCorrect: false},
		{Texte: "Un outil graphique de l'IDE.", EstCorrect: false},
		{Texte: "Un opérateur arithmétique.", EstCorrect: false},
	})
	seedReponses(db, "Quelle est la différence entre un processus et un thread ?", []models.Reponse{
		{Texte: "Il n'y a aucune différence.", EstCorrect: false},
		{Texte: "Un processus a son propre espace mémoire, un thread partage la mémoire du processus parent.", EstCorrect: true},
		{Texte: "Un thread est plus lourd qu'un processus.", EstCorrect: false},
		{Texte: "Un processus ne peut pas communiquer avec d'autres processus.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 17: Architecture logicielle
	// ──────────────────────────────────────────────
	test17 := seedTest(db, models.TestNiveau{
		Titre:       "Architecture logicielle et patrons d'architecture",
		Description: "Évaluation sur les architectures monolithiques, microservices, MVC et REST.",
		Domaine:     "Génie Logiciel",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test17.ID, Enonce: "Quelle est la principale caractéristique d'une architecture microservices ?", Concept: "Architecture logicielle", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test17.ID, Enonce: "Que signifie REST dans le contexte des API web ?", Concept: "Architecture logicielle", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test17.ID, Enonce: "Quel est l'avantage principal d'une architecture en couches (layered) ?", Concept: "Architecture logicielle", Points: 10})

	seedReponses(db, "Quelle est la principale caractéristique d'une architecture microservices ?", []models.Reponse{
		{Texte: "Tout le code est dans un seul binaire exécutable.", EstCorrect: false},
		{Texte: "L'application est décomposée en services indépendants, déployables séparément.", EstCorrect: true},
		{Texte: "Les services partagent tous la même base de données.", EstCorrect: false},
		{Texte: "Il n'y a pas de communication réseau entre les composants.", EstCorrect: false},
	})
	seedReponses(db, "Que signifie REST dans le contexte des API web ?", []models.Reponse{
		{Texte: "Real-time Event Streaming Technology.", EstCorrect: false},
		{Texte: "Representational State Transfer.", EstCorrect: true},
		{Texte: "Remote Execution Service Tool.", EstCorrect: false},
		{Texte: "Relational Entity Storage Transfer.", EstCorrect: false},
	})
	seedReponses(db, "Quel est l'avantage principal d'une architecture en couches (layered) ?", []models.Reponse{
		{Texte: "Elle est plus rapide qu'une architecture microservices.", EstCorrect: false},
		{Texte: "Elle permet la séparation des responsabilités et facilite la maintenance.", EstCorrect: true},
		{Texte: "Elle élimine le besoin de base de données.", EstCorrect: false},
		{Texte: "Elle ne nécessite aucun framework.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 18: Architecture logicielle — Cloud et DevOps
	// ──────────────────────────────────────────────
	test18 := seedTest(db, models.TestNiveau{
		Titre:       "Architecture Cloud et DevOps",
		Description: "Évaluation sur le cloud computing, les conteneurs Docker, CI/CD et l'Infrastructure as Code.",
		Domaine:     "Génie Logiciel",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test18.ID, Enonce: "Qu'est-ce qu'un conteneur Docker ?", Concept: "Architecture logicielle", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test18.ID, Enonce: "Que signifie CI/CD dans le contexte DevOps ?", Concept: "Architecture logicielle", Points: 10})

	seedReponses(db, "Qu'est-ce qu'un conteneur Docker ?", []models.Reponse{
		{Texte: "Une machine virtuelle complète avec son propre OS.", EstCorrect: false},
		{Texte: "Un environnement isolé et léger pour exécuter une application avec toutes ses dépendances.", EstCorrect: true},
		{Texte: "Un outil de gestion de base de données.", EstCorrect: false},
		{Texte: "Un langage de programmation.", EstCorrect: false},
	})
	seedReponses(db, "Que signifie CI/CD dans le contexte DevOps ?", []models.Reponse{
		{Texte: "Code Inspection / Code Delivery.", EstCorrect: false},
		{Texte: "Continuous Integration / Continuous Delivery (ou Deployment).", EstCorrect: true},
		{Texte: "Central Infrastructure / Cloud Deployment.", EstCorrect: false},
		{Texte: "Container Instance / Container Distribution.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// TEST 19: Base de données — NoSQL
	// ──────────────────────────────────────────────
	test19 := seedTest(db, models.TestNiveau{
		Titre:       "Bases de données NoSQL et Big Data",
		Description: "Évaluation sur MongoDB, Redis, les bases document, clé-valeur et les concepts Big Data.",
		Domaine:     "Base de données",
		NiveauCible: "Master 2",
		AuteurID:    authorID,
	})

	seedQuestion(db, models.Question{TestNiveauID: test19.ID, Enonce: "Quel type de base NoSQL stocke des données sous forme de documents JSON ?", Concept: "Base de données", Points: 10})
	seedQuestion(db, models.Question{TestNiveauID: test19.ID, Enonce: "Qu'est-ce que le théorème CAP ?", Concept: "Base de données", Points: 10})

	seedReponses(db, "Quel type de base NoSQL stocke des données sous forme de documents JSON ?", []models.Reponse{
		{Texte: "Base clé-valeur (Redis).", EstCorrect: false},
		{Texte: "Base documentaire (MongoDB).", EstCorrect: true},
		{Texte: "Base en colonnes (Cassandra).", EstCorrect: false},
		{Texte: "Base graphe (Neo4j).", EstCorrect: false},
	})
	seedReponses(db, "Qu'est-ce que le théorème CAP ?", []models.Reponse{
		{Texte: "Un théorème sur le chiffrement des données.", EstCorrect: false},
		{Texte: "Un théorème stipulant qu'un système distribué ne peut garantir simultanément Cohérence, Disponibilité et Tolérance aux partitions.", EstCorrect: true},
		{Texte: "Un algorithme de tri avancé.", EstCorrect: false},
		{Texte: "Un protocole réseau.", EstCorrect: false},
	})

	// ──────────────────────────────────────────────
	// Seed additional Ressources Pédagogiques for all concepts
	// ──────────────────────────────────────────────
	ressources := []models.RessourcePedagogique{
		// UML
		{Titre: "Introduction aux diagrammes de classes UML", Type: "Vidéo", Lien: "https://lms.unchk.sn/ressources/uml_intro.mp4", Concept: "UML", NiveauDifficulte: 1, Pertinence: 5, AuteurID: authorID},
		{Titre: "Exercices pratiques UML : diagrammes de séquence", Type: "Exercice interactif", Lien: "https://lms.unchk.sn/ressources/uml_sequence_exercices.html", Concept: "UML", NiveauDifficulte: 2, Pertinence: 4, AuteurID: authorID},
		// POO
		{Titre: "Les concepts fondamentaux de la Programmation Orientée Objet", Type: "PDF", Lien: "https://lms.unchk.sn/ressources/poo_fondamentaux.pdf", Concept: "POO", NiveauDifficulte: 1, Pertinence: 4, AuteurID: authorID},
		{Titre: "Héritage et polymorphisme en Java", Type: "Vidéo", Lien: "https://lms.unchk.sn/ressources/poo_heritage_java.mp4", Concept: "POO", NiveauDifficulte: 2, Pertinence: 5, AuteurID: authorID},
		// Agile
		{Titre: "Guide pratique Scrum et Agilité", Type: "Exercice interactif", Lien: "https://lms.unchk.sn/ressources/agile_guide.html", Concept: "Agile", NiveauDifficulte: 2, Pertinence: 5, AuteurID: authorID},
		{Titre: "Introduction à la méthode Kanban", Type: "PDF", Lien: "https://lms.unchk.sn/ressources/kanban_intro.pdf", Concept: "Agile", NiveauDifficulte: 1, Pertinence: 4, AuteurID: authorID},
		// SQL
		{Titre: "SQL pour débutants : SELECT, WHERE, JOIN", Type: "Vidéo", Lien: "https://lms.unchk.sn/ressources/sql_debutant.mp4", Concept: "SQL", NiveauDifficulte: 1, Pertinence: 5, AuteurID: authorID},
		{Titre: "Exercices SQL : requêtes avancées", Type: "Exercice interactif", Lien: "https://lms.unchk.sn/ressources/sql_exercices.html", Concept: "SQL", NiveauDifficulte: 2, Pertinence: 4, AuteurID: authorID},
		// Algorithmique
		{Titre: "Algorithmes de tri : cours illustré", Type: "PDF", Lien: "https://lms.unchk.sn/ressources/algo_tri.pdf", Concept: "Algorithmique", NiveauDifficulte: 1, Pertinence: 5, AuteurID: authorID},
		{Titre: "Structures de données : piles, files et arbres", Type: "Vidéo", Lien: "https://lms.unchk.sn/ressources/algo_structures.mp4", Concept: "Algorithmique", NiveauDifficulte: 2, Pertinence: 4, AuteurID: authorID},
		// Réseaux
		{Titre: "Le modèle OSI expliqué simplement", Type: "Vidéo", Lien: "https://lms.unchk.sn/ressources/reseaux_osi.mp4", Concept: "Réseaux", NiveauDifficulte: 1, Pertinence: 5, AuteurID: authorID},
		{Titre: "Protocoles TCP/IP : guide complet", Type: "PDF", Lien: "https://lms.unchk.sn/ressources/reseaux_tcpip.pdf", Concept: "Réseaux", NiveauDifficulte: 2, Pertinence: 4, AuteurID: authorID},
		// Base de données
		{Titre: "Conception de bases de données relationnelles", Type: "PDF", Lien: "https://lms.unchk.sn/ressources/bdd_conception.pdf", Concept: "Base de données", NiveauDifficulte: 1, Pertinence: 5, AuteurID: authorID},
		{Titre: "Normalisation : de la 1NF à la 3NF", Type: "Vidéo", Lien: "https://lms.unchk.sn/ressources/bdd_normalisation.mp4", Concept: "Base de données", NiveauDifficulte: 2, Pertinence: 4, AuteurID: authorID},
		// HTML/CSS
		{Titre: "HTML5 et sémantique web", Type: "PDF", Lien: "https://lms.unchk.sn/ressources/html5_semantique.pdf", Concept: "HTML/CSS", NiveauDifficulte: 1, Pertinence: 5, AuteurID: authorID},
		{Titre: "Flexbox et Grid CSS : layout moderne", Type: "Vidéo", Lien: "https://lms.unchk.sn/ressources/css_flexbox_grid.mp4", Concept: "HTML/CSS", NiveauDifficulte: 2, Pertinence: 4, AuteurID: authorID},
		// Programmation
		{Titre: "Les bases de la programmation : variables et boucles", Type: "Vidéo", Lien: "https://lms.unchk.sn/ressources/prog_bases.mp4", Concept: "Programmation", NiveauDifficulte: 1, Pertinence: 5, AuteurID: authorID},
		{Titre: "Fonctions et modularité en programmation", Type: "PDF", Lien: "https://lms.unchk.sn/ressources/prog_fonctions.pdf", Concept: "Programmation", NiveauDifficulte: 2, Pertinence: 4, AuteurID: authorID},
		// Architecture logicielle
		{Titre: "Introduction aux architectures logicielles", Type: "PDF", Lien: "https://lms.unchk.sn/ressources/archi_intro.pdf", Concept: "Architecture logicielle", NiveauDifficulte: 1, Pertinence: 5, AuteurID: authorID},
		{Titre: "Microservices vs Monolithique : guide comparatif", Type: "Vidéo", Lien: "https://lms.unchk.sn/ressources/archi_microservices.mp4", Concept: "Architecture logicielle", NiveauDifficulte: 2, Pertinence: 4, AuteurID: authorID},
	}
	for _, r := range ressources {
		db.FirstOrCreate(&r, models.RessourcePedagogique{Titre: r.Titre})
	}

	// Seed system configurations
	defaultConfigs := []models.SystemConfig{
		{Key: "seuil_recommandation", Value: "60", Description: "Seuil de réussite (%) en dessous duquel un concept est considéré comme une lacune (recommandation de ressources de remédiation)."},
		{Key: "ai_engine_url", Value: "http://localhost:8000/api/recommend", Description: "URL de l'API de l'Agent IA Python pour la génération des recommandations."},
	}
	for _, configEntry := range defaultConfigs {
		db.FirstOrCreate(&configEntry, models.SystemConfig{Key: configEntry.Key})
	}

	log.Println("Données contextuelles UNCHK injectées avec succès (19 tests, 10 concepts, 20 ressources, 2 configurations).")
}

// Ensure admin user exists (for migration from old seeds)
func init() {
	// This is handled by seedUser function which checks for non-bcrypt passwords
}
