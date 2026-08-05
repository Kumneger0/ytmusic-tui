# Dockerfile for Python Connect RPC Server

# --- Builder Stage ---
FROM python:3.13-slim AS builder

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    UV_PROJECT_ENVIRONMENT="/app/.venv"

WORKDIR /app

# Copy uv binary for fast package installation
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

# Copy dependency manifests
COPY pyproject.toml uv.lock ./

# Install dependencies into virtual environment (.venv)
RUN uv sync --frozen --no-install-project --no-dev

# Copy project source and proto definitions
COPY . .

# Generate Python Protobuf & Connect RPC files
RUN mkdir -p grpc_server/gen && \
    uv run --no-dev python -m grpc_tools.protoc \
        -Iproto \
        --python_out=grpc_server/gen \
        --pyi_out=grpc_server/gen \
        --plugin=protoc-gen-connect-python=/app/.venv/bin/protoc-gen-connect-python \
        --connect-python_out=grpc_server/gen \
        proto/music.proto && \
    sed -i 's/^import music_pb2 as/from . import music_pb2 as/' grpc_server/gen/music_connect.py && \
    sed -i 's/from connectrpc.compression import Compression/from connectrpc.codec import Codec\nfrom connectrpc.compression import Compression/' grpc_server/gen/music_connect.py && \
    sed -i 's/compressions: Iterable\[Compression\] | None = None) -> None:/compressions: Iterable[Compression] | None = None, codecs: Iterable[Codec] | None = None) -> None:/' grpc_server/gen/music_connect.py && \
    sed -i 's/compressions=compressions,/compressions=compressions,\n            codecs=codecs,/' grpc_server/gen/music_connect.py

# --- Runtime Stage ---
FROM python:3.13-slim AS runner

WORKDIR /app

ENV PYTHONUNBUFFERED=1 \
    PYTHONPATH=/app \
    PORT=8080 \
    PATH="/app/.venv/bin:$PATH"

# Copy virtual environment and app files from builder
COPY --from=builder /app/.venv /app/.venv
COPY --from=builder /app/grpc_server /app/grpc_server
COPY --from=builder /app/proto /app/proto

# Expose default port
EXPOSE 8080

# Run the Connect RPC server
CMD ["/app/.venv/bin/python", "grpc_server/main.py"]
