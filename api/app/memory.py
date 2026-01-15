from mem0 import Memory

# openai provider routes to openrouter
config = {
    "vector_store": {
        "provider": "qdrant",
        "config": {"host": "localhost", "port": 6333},
    },
    "llm": {
        "provider": "openai",
        "config": {"model": "google/gemini-3-flash-preview", "temperature": 0.1},
    },
    "embedder": {
        "provider": "openai",
        "config": {"model": "text-embedding-3-small"},
    },
    "reranker": {
        "provider": "sentence_transformer",
        "config": {"model": "cross-encoder/ms-marco-MiniLM-L-6-v2"},
    },
}

memory = Memory.from_config(config)
