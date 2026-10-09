#!/bin/bash
set -e

SERVICE_NAME=${1:-"all"}

echo "================================================="
echo " Deploying service: $SERVICE_NAME via Docker Compose"
echo "================================================="

# 1. Determine repository directory
REPO_DIR="/home/barath/Codes/penne-server"
if [ ! -d "$REPO_DIR" ]; then
  REPO_DIR="/home/barath/Codes/go-code"
fi

if [ ! -d "$REPO_DIR" ]; then
  echo "Error: Repository directory not found."
  exit 1
fi

cd "$REPO_DIR"

# 2. Pull latest code
echo "Pulling latest code in $REPO_DIR..."
git fetch origin main
git reset --hard origin/main

# 3. Load environment variables from penne-service .env
if [ -f "$REPO_DIR/src/penne-service/.env" ]; then
  echo "Loading environment from $REPO_DIR/src/penne-service/.env..."
  set -a
  source "$REPO_DIR/src/penne-service/.env"
  set +a
elif [ -f "$REPO_DIR/.env" ]; then
  echo "Loading environment from $REPO_DIR/.env..."
  set -a
  source "$REPO_DIR/.env"
  set +a
fi

# 4. Deploy via Docker Compose
case "$SERVICE_NAME" in
  penne-service)
    echo "Rebuilding and restarting penne-service..."
    docker compose up -d --build penne-service
    ;;
  cadence-penne-service|cadence-worker)
    echo "Rebuilding and restarting cadence-worker..."
    docker compose up -d --build cadence-worker
    ;;
  gateway)
    echo "Rebuilding and restarting gateway..."
    docker compose up -d --build gateway
    ;;
  all|"")
    echo "Rebuilding and deploying entire backend stack..."
    docker compose up -d --build
    ;;
  *)
    echo "Deploying target service: $SERVICE_NAME..."
    docker compose up -d --build "$SERVICE_NAME"
    ;;
esac

# 5. Clean up dangling images to conserve disk space
docker image prune -f

# 6. Verify deployment health
echo "Waiting for services to become healthy..."
sleep 3
if curl -s -f http://localhost:8080/health > /dev/null; then
  echo " Health check passed (Gateway & Downstream healthy)!"
else
  echo "⚠️ Warning: Health check did not return 200 OK immediately. Inspecting container logs:"
  docker compose logs --tail 20
fi

docker compose ps

echo "================================================="
echo " Successfully deployed $SERVICE_NAME!"
echo "================================================="
