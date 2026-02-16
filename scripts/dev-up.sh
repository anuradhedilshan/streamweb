#!/usr/bin/env bash
set -euo pipefail

docker compose -f docker-compose.dev.yml up -d --build

echo "Dev stack up: api, postgres, redis, minio, nginx"
echo "API: http://127.0.0.1:8080"
echo "MinIO console: http://127.0.0.1:9001 (minio/minio123)"
echo "NGINX gateway: http://127.0.0.1:8088"
