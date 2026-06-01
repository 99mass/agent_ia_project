from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import pandas as pd
import numpy as np
from sqlalchemy import create_engine
import os

app = FastAPI(title="AI Agent - Recommandation UNCHK")

# Database connection
DB_USER = os.getenv("DB_USER", "unchk_admin")
DB_PASSWORD = os.getenv("DB_PASSWORD", "unchk_password")
DB_HOST = os.getenv("DB_HOST", "localhost")
DB_NAME = os.getenv("DB_NAME", "unchk_agent_ia")

DATABASE_URL = f"postgresql://{DB_USER}:{DB_PASSWORD}@{DB_HOST}:5432/{DB_NAME}"
engine = create_engine(DATABASE_URL)

class RecommendationRequest(BaseModel):
    resultat_id: int
    etudiant_id: int
    student_scores: dict  # e.g., {"UML": 40, "POO": 80, "Agile": 30}
    format_prefere: str = "tous"
    seuil_reussite: int = None

@app.post("/api/recommend")
def generate_recommendations(req: RecommendationRequest):
    try:
        # Load mastery threshold (fallback: database configuration, default 60.0)
        mastery_threshold = 60.0
        
        if req.seuil_reussite is not None and req.seuil_reussite > 0:
            mastery_threshold = float(req.seuil_reussite)
            print(f"Utilisation du seuil spécifique au test défini par l'enseignant : {mastery_threshold}%")
        else:
            try:
                config_query = "SELECT value FROM system_configs WHERE key = 'seuil_recommandation' LIMIT 1"
                df_config = pd.read_sql(config_query, engine)
                if not df_config.empty:
                    mastery_threshold = float(df_config.iloc[0]['value'])
                    print(f"Utilisation du seuil système global : {mastery_threshold}%")
            except Exception as ce:
                print("Erreur de chargement du seuil global, utilisation de 60.0 :", ce)

        # Load resources from database into a pandas DataFrame
        query = "SELECT * FROM ressource_pedagogiques"
        df_resources = pd.read_sql(query, engine)
        
        if df_resources.empty:
            return {"recommendations": []}

        recommendations = []
        
        # Analyze student scores
        for concept, score in req.student_scores.items():
            concept_res = df_resources[df_resources['concept'] == concept]
            if concept_res.empty:
                continue

            if score < mastery_threshold:
                # If the score is below the mastery threshold, recommend remedial resources (niveau_difficulte <= 2)
                suitable_res = concept_res[concept_res['niveau_difficulte'] <= 2]
                if suitable_res.empty:
                    suitable_res = concept_res
                
                # Apply format preference filter if active and matches resources
                if req.format_prefere and req.format_prefere != "tous":
                    pref_filter = req.format_prefere.lower()
                    if pref_filter == "video":
                        format_res = suitable_res[suitable_res['type'].str.lower().str.contains("video|vidéo")]
                    elif pref_filter == "pdf":
                        format_res = suitable_res[suitable_res['type'].str.lower().str.contains("pdf|cours")]
                    elif pref_filter == "exercice":
                        format_res = suitable_res[suitable_res['type'].str.lower().str.contains("exercice|interactif|tp|pratique")]
                    else:
                        format_res = suitable_res[suitable_res['type'].str.lower().str.contains(pref_filter)]
                    
                    if not format_res.empty:
                        suitable_res = format_res

                # Sort by pertinence (highest first) and pick top 2
                top_res = suitable_res.sort_values(by='pertinence', ascending=False).head(2)
                for _, row in top_res.iterrows():
                    recommendations.append({
                        "ressource_id": row['id'],
                        "titre": row['titre'],
                        "lien": row['lien'],
                        "concept": concept,
                        "raison": f"Lacune identifiée en {concept} (Score: {score}%, seuil de maîtrise: {mastery_threshold}%)"
                    })
            else:
                # If the score is >= mastery_threshold, recommend advanced resources for deepening knowledge (niveau_difficulte >= 2)
                suitable_res = concept_res[concept_res['niveau_difficulte'] >= 2]
                if suitable_res.empty:
                    suitable_res = concept_res
                
                # Apply format preference filter if active and matches resources
                if req.format_prefere and req.format_prefere != "tous":
                    pref_filter = req.format_prefere.lower()
                    if pref_filter == "video":
                        format_res = suitable_res[suitable_res['type'].str.lower().str.contains("video|vidéo")]
                    elif pref_filter == "pdf":
                        format_res = suitable_res[suitable_res['type'].str.lower().str.contains("pdf|cours")]
                    elif pref_filter == "exercice":
                        format_res = suitable_res[suitable_res['type'].str.lower().str.contains("exercice|interactif|tp|pratique")]
                    else:
                        format_res = suitable_res[suitable_res['type'].str.lower().str.contains(pref_filter)]
                    
                    if not format_res.empty:
                        suitable_res = format_res

                # Pick the top 1 most relevant resource for enrichment
                top_res = suitable_res.sort_values(by='pertinence', ascending=False).head(1)
                for _, row in top_res.iterrows():
                    recommendations.append({
                        "ressource_id": row['id'],
                        "titre": row['titre'],
                        "lien": row['lien'],
                        "concept": concept,
                        "raison": f"Approfondissement conseillé en {concept} (Score: {score}%, seuil de maîtrise: {mastery_threshold}%)"
                    })
        
        # In a real scenario, we should also save these recommendations into the `recommandations` table
        # using SQLAlchemy, but returning them to the backend is the first step.
        
        return {"recommendations": recommendations}

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
