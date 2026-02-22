#!/usr/bin/env bash
set -euo pipefail

MODEL_ID=""
INCLUDE_PATTERN="*Q8_0*.gguf"

usage() {
  echo "Usage: $0 [--include <glob>] <org/model>"
  echo ""
  echo "Examples:"
  echo "  $0 unsloth/Qwen3-Next-80B-A3B-Thinking-GGUF"
  echo "  $0 --include \"*Q8_0*.gguf\" unsloth/Qwen3-Next-80B-A3B-Thinking-GGUF"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --include)
      shift
      if [[ $# -eq 0 ]]; then
        echo "Error: --include requires a value" >&2
        usage
        exit 1
      fi
      INCLUDE_PATTERN="$1"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      if [[ -z "$MODEL_ID" ]]; then
        MODEL_ID="$1"
      else
        echo "Error: unknown parameter: $1" >&2
        usage
        exit 1
      fi
      ;;
  esac
  shift
done

if [[ -z "$MODEL_ID" ]]; then
  echo "Error: model id is required" >&2
  usage
  exit 1
fi

if ! command -v uvx >/dev/null 2>&1; then
  echo "Error: uvx command not found in PATH" >&2
  echo "Install uv and ensure ~/.local/bin is in PATH." >&2
  exit 1
fi

echo "Downloading $MODEL_ID with include pattern: $INCLUDE_PATTERN"
exec uvx hf download "$MODEL_ID" --include "$INCLUDE_PATTERN"
