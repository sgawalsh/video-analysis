from fastapi import FastAPI # type: ignore
from sentence_transformers import SentenceTransformer # type: ignore
import faiss # type: ignore
import numpy as np # type: ignore

app = FastAPI()

model = SentenceTransformer("BAAI/bge-small-en-v1.5")

prefix = "Represent this sentence for searching relevant passages: "

@app.post("/embedAndSearch")
def embedAndSearch(payload: dict):
    texts = payload["texts"]
    # print(f"Embedding {len(texts)} texts")
    # print(f"Texts: {texts[:5]}")
    embeddings = model.encode(texts, normalize_embeddings=True)

    # print(embeddings.shape)
    
    index = faiss.IndexFlatIP(384)  # Assuming 384-dimensional embeddings

    index.add(np.array(embeddings, dtype=np.float32))

    # print(f"Query: {payload['query']}")

    query_embedding = model.encode([prefix + payload['query']], normalize_embeddings=True)

    distances, indices = index.search(np.array([query_embedding[0]], dtype=np.float32), k=5)

    # print(f"Distances: {distances}\nIndices: {indices}")

    return {"distances": distances.flatten().tolist(), "indices": indices.flatten().tolist()}