#!/bin/bash
set -e

# Enable BuildKit for fast compilation and cache mounts
export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

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

# 3. Load environment variables from service-wise .env files
if [ -f "$REPO_DIR/src/penne-service/.env" ]; then
  echo "Loading environment from $REPO_DIR/src/penne-service/.env..."
  set -a
  source "$REPO_DIR/src/penne-service/.env"
  set +a
fi

if [ -f "$REPO_DIR/src/nutrition-service/.env" ]; then
  echo "Loading environment from $REPO_DIR/src/nutrition-service/.env..."
  set -a
  source "$REPO_DIR/src/nutrition-service/.env"
  set +a
fi

# 4. Deploy via Docker Compose
case "$SERVICE_NAME" in
  penne-service)
    echo "Rebuilding and restarting penne-service..."
    docker compose build penne-service
    docker compose up -d penne-service cadence-worker
    ;;
  cadence-penne-service|cadence-worker)
    echo "Rebuilding and restarting cadence-worker..."
    docker compose build penne-service
    docker compose up -d cadence-worker
    ;;
  gateway)
    echo "Rebuilding and restarting gateway..."
    docker compose build gateway
    docker compose up -d gateway
    ;;
  all|"")
    echo "Starting database service..."
    docker compose up -d db
    sleep 2
    # Ensure dedicated database exists for nouri/nutrition-service
    docker exec -i penne_db psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'nouri_db'" 2>/dev/null | grep -q 1 || \
    docker exec -i penne_db psql -U postgres -c "CREATE DATABASE nouri_db;" 2>/dev/null || true
    echo "Rebuilding backend services (penne-service, nutrition-service, gateway)..."
    docker compose build penne-service
    docker compose build nutrition-service
    docker compose build gateway
    echo "Deploying entire backend stack..."
    docker compose up -d
    ;;
  nutrition-service)
    echo "Starting database service..."
    docker compose up -d db
    sleep 2
    docker exec -i penne_db psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'nouri_db'" 2>/dev/null | grep -q 1 || \
    docker exec -i penne_db psql -U postgres -c "CREATE DATABASE nouri_db;" 2>/dev/null || true
    echo "Rebuilding and restarting nutrition-service..."
    docker compose build nutrition-service
    docker compose up -d nutrition-service
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
