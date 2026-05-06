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

@app.post("/api/recommend")
def generate_recommendations(req: RecommendationRequest):
    try:
        # Load resources from database into a pandas DataFrame
        query = "SELECT * FROM ressource_pedagogiques"
        df_resources = pd.read_sql(query, engine)
        
        if df_resources.empty:
            return {"recommendations": []}

        recommendations = []
        
        # Analyze student scores
        for concept, score in req.student_scores.items():
            # If the score is below the mastery threshold (e.g. 60%)
            if score < 60:
                # Filter resources by concept
                concept_res = df_resources[df_resources['concept'] == concept]
                
                # Further filter for beginner/remedial resources (niveau_difficulte <= 2)
                suitable_res = concept_res[concept_res['niveau_difficulte'] <= 2]
                
                # Sort by pertinence (highest first) and pick top 2
                top_res = suitable_res.sort_values(by='pertinence', ascending=False).head(2)
                
                for _, row in top_res.iterrows():
                    recommendations.append({
                        "ressource_id": row['id'],
                        "titre": row['titre'],
                        "lien": row['lien'],
                        "concept": concept,
                        "raison": f"Lacune identifiée en {concept} (Score: {score}%)"
                    })
        
        # In a real scenario, we should also save these recommendations into the `recommandations` table
        # using SQLAlchemy, but returning them to the backend is the first step.
        
        return {"recommendations": recommendations}

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
