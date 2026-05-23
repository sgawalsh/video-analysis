from fastapi import FastAPI # type: ignore
from sentence_transformers import SentenceTransformer # type: ignore

app = FastAPI()

model = SentenceTransformer("BAAI/bge-small-en-v1.5")

@app.post("/embed")
def embed(payload: dict):
    texts = payload["texts"]
    vectors = model.encode(texts, normalize_embeddings=True)
    return {"embeddings": vectors.tolist()}