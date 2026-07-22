from fastapi import FastAPI # type: ignore
from sentence_transformers import SentenceTransformer # type: ignore
import faiss # type: ignore
import numpy as np # type: ignore
import requests # type: ignore

app = FastAPI()

model = SentenceTransformer("BAAI/bge-small-en-v1.5")

prefix = "Represent this sentence for searching relevant passages: "
windowSize = 2

@app.get("/health")
def health():
    return {"status": "ok"}

@app.post("/embedAndSearch")
def embedAndSearch(payload: dict):
    texts = payload["Texts"]
    # print(f"Embedding {len(texts)} texts")
    # print(f"Texts: {texts[:5]}")
    embeddings = model.encode(texts, normalize_embeddings=True)

    # print(embeddings.shape)
    
    index = faiss.IndexFlatIP(384)  # Assuming 384-dimensional embeddings

    index.add(np.array(embeddings, dtype=np.float32))

    # print(f"Query: {payload['Query']}")

    query_embedding = model.encode([prefix + payload['Query']], normalize_embeddings=True)

    distances, indices = index.search(np.array([query_embedding[0]], dtype=np.float32), k=5)

    # print(f"Distances: {distances}\nIndices: {indices}")

    return {"distances": distances.flatten().tolist(), "indices": indices.flatten().tolist()}

@app.post("/getTopicWindows")
def getTopicWindows(payload: dict):
    texts = payload["Texts"]
    similarities = windowEmbeddingsToSimilarities(model.encode(texts, normalize_embeddings=True))

    boundaries = [0]
    threshold = np.percentile(similarities, 10)

    # if similarities[0] < threshold and similarities[0] < similarities[1]: # Find local minima in similarities
    #     boundaries.append(windowSize)

    for i in range(1, len(similarities) - 1):
        if (
            similarities[i] < similarities[i - 1]
            and similarities[i] < similarities[i + 1]
            and similarities[i] < threshold
        ):
            boundaries.append(i + windowSize)
    
    # if similarities[len(similarities) - 1] < threshold and similarities[len(similarities) - 1] < similarities[len(similarities) - 2]: # Find local minima in similarities
    #     boundaries.append(len(similarities) - 1 + windowSize)

    segments = []
    for low, high in zip(boundaries[:-1], boundaries[1:]):
        segments.append(" ".join(texts[low:high]))

    segments.append(" ".join(texts[boundaries[-1]:]))

    return {
        "boundaries": boundaries,
        "chapterTexts": segments
    }

def windowEmbeddingsToSimilarities(embeddings):
    windowEmbeddings = embeddingsToWindowEmbeddings(embeddings)
    similarities = []

    for i in range(len(windowEmbeddings) - windowSize):
        sim = np.dot(
            windowEmbeddings[i],
            windowEmbeddings[i + windowSize]
        )

        similarities.append(sim)
    
    return similarities

def embeddingsToWindowEmbeddings(embeddings):
    windowEmbeddings = []
    for i in range(len(embeddings) - windowSize + 1):
        window = embeddings[i:i + windowSize]
        avgEmbedding = np.mean(window, axis=0)
        avgEmbedding /= np.linalg.norm(avgEmbedding)
        windowEmbeddings.append(avgEmbedding)

    return np.array(windowEmbeddings)