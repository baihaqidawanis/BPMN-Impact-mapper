import json
import sys

from sentence_transformers import SentenceTransformer


def main():
    model_name = sys.argv[1]
    request = json.load(sys.stdin)
    texts = request.get("texts", [])
    if not isinstance(texts, list) or not all(isinstance(value, str) for value in texts):
        raise ValueError("texts must be a list of strings")
    model = SentenceTransformer(model_name, local_files_only=True, device="cpu")
    vectors = model.encode(
        texts,
        batch_size=32,
        convert_to_numpy=True,
        normalize_embeddings=True,
        show_progress_bar=False,
    )
    json.dump(
        {
            "dimension": int(model.get_sentence_embedding_dimension()),
            "vectors": [[round(float(value), 8) for value in vector] for vector in vectors],
        },
        sys.stdout,
        separators=(",", ":"),
    )


if __name__ == "__main__":
    main()
