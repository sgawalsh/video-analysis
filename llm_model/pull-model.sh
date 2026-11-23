#!/bin/sh
set -eu

MODEL="${OLLAMA_MODEL:-llama3.2:1b}"

ollama serve &
OLLAMA_PID=$!

cleanup() {
    kill "$OLLAMA_PID" 2>/dev/null || true
    wait "$OLLAMA_PID" 2>/dev/null || true
}

trap cleanup EXIT
trap 'exit 143' TERM
trap 'exit 130' INT

# Wait for Ollama to become available
until ollama list >/dev/null 2>&1; do
    if ! kill -0 "$OLLAMA_PID" 2>/dev/null; then
        echo "Ollama server exited unexpectedly"
        exit 1
    fi
    sleep 1
done

# Check whether the model already exists
if ollama list | awk -v model="$MODEL" \
    'NR > 1 && $1 == model { found=1 } END { exit !found }'; then
    echo "Model already exists: $MODEL"
    exit 0
fi

# Download model
echo "Downloading model: $MODEL"
ollama pull "$MODEL"

# Verify successful download
if ! ollama list | awk -v model="$MODEL" \
    'NR > 1 && $1 == model { found=1 } END { exit !found }'; then
    echo "Model download completed but model was not found"
    exit 1
fi

echo "Model successfully provisioned: $MODEL"