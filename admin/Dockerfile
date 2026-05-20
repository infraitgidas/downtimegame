# =============================================================================
# Admin Panel — Downtime Game
# Multi-stage: build React+Vite, then serve with nginx
# =============================================================================

# --- Builder ---
FROM node:24-alpine AS builder

WORKDIR /app

# Copy manifest first for layer caching
COPY package.json ./

# npm install with workarounds for overlay2 bug + SSL cert fix:
# - Install ca-certificates: Alpine no incluye CAs, necesario para SSL
# - --mount=type=cache: BuildKit cache mount, persiste caché entre builds
#   (baja de 500s a <10s en rebuilds)
# - NPM_CONFIG_CACHE: apunta el caché al mount point persistente
# - --no-optional: reduce package count
# - --no-audit --no-fund: skip network checks
# - --no-package-lock: avoid lockfile format issues
ENV NPM_CONFIG_CACHE=/tmp/.npm
RUN --mount=type=cache,target=/tmp/.npm \
    apk add --no-cache ca-certificates && \
    update-ca-certificates && \
    npm install --no-optional --no-audit --no-fund --no-package-lock --loglevel=error

COPY . .
RUN npm run build

# --- Runtime ---
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx-admin.conf /etc/nginx/conf.d/default.conf

EXPOSE 5174
CMD ["nginx", "-g", "daemon off;"]
