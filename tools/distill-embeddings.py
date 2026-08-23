#!/usr/bin/env python3
"""
One-time distillation export for hero-embed-v1.

Downloads minishlab/potion-base-8M (distilled from BAAI/bge-base-en-v1.5,
256-dim) from HuggingFace and exports vocab.txt + weights.bin + config.json
in the format the Go embedding engine expects.

By default, prunes dead vocabulary entries (BERT special tokens and ##
subword tokens that Hero's whole-word tokenizer will never produce).

Requirements (install in a disposable venv):
    pip install huggingface_hub safetensors numpy

Usage:
    python3 tools/distill-embeddings.py [output_dir]
    python3 tools/distill-embeddings.py --no-prune [output_dir]

Default output: internal/embeddings/defaultmodel/
"""

import json
import os
import re
import sys

import numpy as np
from huggingface_hub import hf_hub_download
from safetensors.numpy import load_file


REPO_ID = "minishlab/potion-base-8M"
REVISION = "bf8b056651a2c21b8d2565580b8569da283cab23"

# Tokens that Hero's Go tokenizer will never produce.
DEAD_PREFIXES = {"##", "[PAD]", "[UNK]", "[CLS]", "[SEP]", "[MASK]"}


def load_vocab(tokenizer_path):
    """Extract ordered vocabulary from tokenizer.json."""
    with open(tokenizer_path) as f:
        data = json.load(f)

    vocab = data["model"]["vocab"]
    tokens = sorted(vocab.items(), key=lambda x: x[1])
    return [t for t, _ in tokens]


def is_dead_token(token):
    """Return True if Hero's tokenizer will never produce this token."""
    if token in DEAD_PREFIXES:
        return True
    if token.startswith("##"):
        return True
    return False


def prune(tokens, embeddings):
    """Remove dead tokens and their corresponding weight rows."""
    keep = [(i, t) for i, t in enumerate(tokens) if not is_dead_token(t)]
    pruned_count = len(tokens) - len(keep)

    indices = [i for i, _ in keep]
    pruned_tokens = [t for _, t in keep]
    pruned_embeddings = embeddings[indices]

    return pruned_tokens, pruned_embeddings, pruned_count


def export(tokens, embeddings, output_dir):
    """Write vocab.txt, weights.bin, config.json in the Go loader format."""
    os.makedirs(output_dir, exist_ok=True)
    dim = embeddings.shape[1]

    vocab_path = os.path.join(output_dir, "vocab.txt")
    with open(vocab_path, "w") as f:
        for token in tokens:
            f.write(token + "\n")

    weights_path = os.path.join(output_dir, "weights.bin")
    flat = embeddings.astype(np.float32).flatten()
    with open(weights_path, "wb") as f:
        f.write(flat.tobytes())

    config_path = os.path.join(output_dir, "config.json")
    with open(config_path, "w") as f:
        json.dump({"dim": dim, "vocab_size": len(tokens)}, f, indent=2)

    print(f"Exported {len(tokens)} tokens x {dim} dims")
    print(f"  vocab.txt:   {os.path.getsize(vocab_path):,} bytes")
    print(f"  weights.bin: {os.path.getsize(weights_path):,} bytes")
    print(f"  config.json: {os.path.getsize(config_path):,} bytes")
    total = sum(
        os.path.getsize(os.path.join(output_dir, f))
        for f in ["vocab.txt", "weights.bin", "config.json"]
    )
    print(f"  total:       {total:,} bytes ({total / 1048576:.1f} MB)")


def main():
    do_prune = True
    args = [a for a in sys.argv[1:] if a != "--no-prune"]
    if "--no-prune" in sys.argv:
        do_prune = False

    script_dir = os.path.dirname(os.path.abspath(__file__))
    repo_root = os.path.dirname(script_dir)
    default_output = os.path.join(
        repo_root, "internal", "embeddings", "defaultmodel"
    )

    output_dir = args[0] if args else default_output

    print(f"Downloading {REPO_ID} from HuggingFace...")
    tokenizer_path = hf_hub_download(
        REPO_ID, "tokenizer.json", revision=REVISION
    )
    weights_path = hf_hub_download(
        REPO_ID, "model.safetensors", revision=REVISION
    )

    try:
        config_path = hf_hub_download(
            REPO_ID, "config.json", revision=REVISION
        )
        with open(config_path) as f:
            model_config = json.load(f)
        print(f"Model config: {json.dumps(model_config, indent=2)}")
    except Exception:
        pass

    print("Loading vocabulary...")
    tokens = load_vocab(tokenizer_path)
    print(f"  Vocabulary size: {len(tokens)}")

    print("Loading weight matrix...")
    tensors = load_file(weights_path)
    print(f"  Tensor keys: {list(tensors.keys())}")

    emb_key = None
    for key in tensors:
        if "embed" in key.lower() or "weight" in key.lower():
            emb_key = key
            break
    if emb_key is None:
        emb_key = list(tensors.keys())[0]

    embeddings = tensors[emb_key]
    print(f"  Embedding shape: {embeddings.shape} (key={emb_key})")

    if embeddings.shape[0] != len(tokens):
        print(
            f"  WARNING: vocab size ({len(tokens)}) != "
            f"embedding rows ({embeddings.shape[0]})"
        )
        n = min(len(tokens), embeddings.shape[0])
        tokens = tokens[:n]
        embeddings = embeddings[:n]
        print(f"  Truncated to {n} entries")

    if do_prune:
        print("\nPruning dead tokens (## subwords, BERT specials)...")
        tokens, embeddings, pruned_count = prune(tokens, embeddings)
        print(f"  Pruned {pruned_count} dead tokens, {len(tokens)} remaining")

    print(f"\nExporting to {output_dir}...")
    export(tokens, embeddings, output_dir)
    print("Done.")


if __name__ == "__main__":
    main()
