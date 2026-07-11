#!/usr/bin/env bash
set -euo pipefail

APP_DIR=/opt/apps/fmert-blog
SOURCE_DIR="$APP_DIR/source"
cd "$SOURCE_DIR"
git fetch --prune origin main
REMOTE_REVISION=$(git rev-parse origin/main)
DEPLOYED_REVISION=$(cat "$APP_DIR/.deployed-revision" 2>/dev/null || true)

healthy() {
  [[ $(docker inspect --format '{{.State.Health.Status}}' "$1" 2>/dev/null || true) == healthy ]]
}

if [[ "$REMOTE_REVISION" == "$DEPLOYED_REVISION" ]] && healthy fmert_blog && healthy fmert_story_admin; then
  exit 0
fi

git checkout main
git reset --hard origin/main
git submodule sync --recursive
git submodule update --init --recursive

caddy validate --config deploy/Caddyfile
docker compose --env-file deploy/.env -f deploy/docker-compose.yml build --pull
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up -d --remove-orphans
install -m 0644 deploy/Caddyfile /etc/caddy/Caddyfile
systemctl reload caddy

for _ in {1..30}; do
  if healthy fmert_blog && healthy fmert_story_admin; then
    printf '%s\n' "$REMOTE_REVISION" > "$APP_DIR/.deployed-revision"
    docker image prune -f --filter 'dangling=true'
    curl --fail --silent --show-error http://127.0.0.1:8080/ >/dev/null
    curl --fail --silent --show-error http://127.0.0.1:8081/health >/dev/null
    exit 0
  fi
  sleep 2
done

echo 'Blog containers did not become healthy' >&2
exit 1
