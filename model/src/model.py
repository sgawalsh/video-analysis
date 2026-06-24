from fastapi import FastAPI # type: ignore
from sentence_transformers import SentenceTransformer # type: ignore
import faiss # type: ignore
import numpy as np # type: ignore
import requests # type: ignore

app = FastAPI()

model = SentenceTransformer("BAAI/bge-small-en-v1.5")
ollamaUrl = "http://ollama:11434/api/generate"

prefix = "Represent this sentence for searching relevant passages: "
windowSize = 2

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
        segments.append(texts[low:high])

    segments.append(texts[boundaries[-1]:])
    segments = [" ".join(seg) for seg in segments]

    segmentedChapters = []

    for idx, segment_text in enumerate(segments):
        truncated_text = " ".join(segment_text.split()[:300]) 
        
        prompt = (
            f"Write a short, descriptive video chapter title (under 5 words) for this transcript segment. Respond with just the title, do not include introductory words or punctuation. Text: {truncated_text}"
        )
        
        try:
            payload = {
                "model": "llama3.2:1b",
                "prompt": prompt,
                "stream": False,
                "options": {
                    "num_predict": 10,   # Hard token limit saves CPU cycles per chunk
                    "temperature": 0.1
                }
            }
            
            response = requests.post(ollamaUrl, json=payload, timeout=500)
            title = response.json().get("response", f"Chapter {idx + 1}").strip()
        except Exception:
            title = f"Chapter {idx + 1}" # Fallback if the LLM times out under load
            print("Timed out!")
            
        segmentedChapters.append({
            "ID": idx,
            "Title": title,
            "Text": segment_text
        })

        print(f"Title: {title}")

    return {
        "boundaries": boundaries,
        "chapters": segmentedChapters
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

def generate_section_title(transcript_chunk: str) -> str:
    
    system_prompt = "You are an expert video editor. Analyze the following video transcript segment and generate exactly ONE informative, concise title (under 7 words) that accurately describes the topic covered. Return only the title. Do not include quotes, pleasantries, or explanations."
    
    payload = {
        "model": "llama3", # Or mistral, phi3, etc.
        "prompt": f"{system_prompt}\n\nTranscript Segment:\n{transcript_chunk}",
        "stream": False
    }
    
    response = requests.post(ollamaUrl, json=payload)
    result = response.json()
    return result.get("response", "").strip()