import json
import hashlib
import platform
import sys

import sentence_transformers
from sentence_transformers import SentenceTransformer


def model_artifact_sha256(model, model_name):
    digest = hashlib.sha256()
    digest.update(model_name.encode("utf-8"))
    digest.update(sentence_transformers.__version__.encode("ascii"))
    digest.update(str(model.get_sentence_embedding_dimension()).encode("ascii"))
    digest.update(str(model.max_seq_length).encode("ascii"))
    for module in model:
        config_reader = getattr(module, "get_config_dict", None)
        if callable(config_reader):
            digest.update(json.dumps(config_reader(), sort_keys=True, separators=(",", ":"), default=str).encode("utf-8"))
    for name, tensor in sorted(model.state_dict().items()):
        digest.update(name.encode("utf-8"))
        digest.update(b"\0")
        value = tensor.detach().cpu().contiguous()
        digest.update(str(value.dtype).encode("ascii"))
        digest.update(str(tuple(value.shape)).encode("ascii"))
        digest.update(value.numpy().tobytes())
    tokenizer = getattr(model[0], "tokenizer", None)
    if tokenizer is not None:
        digest.update(json.dumps(tokenizer.get_vocab(), sort_keys=True, separators=(",", ":")).encode("utf-8"))
    return digest.hexdigest()


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
            "runtime": {
                "python_version": platform.python_version(),
                "sentence_transformers_version": sentence_transformers.__version__,
                "model_artifact_sha256": model_artifact_sha256(model, model_name),
            },
            "vectors": [[round(float(value), 8) for value in vector] for vector in vectors],
        },
        sys.stdout,
        separators=(",", ":"),
    )


if __name__ == "__main__":
    main()
