# =============================================================================
# Admin Panel — Downtime Game
# Multi-stage: build React+Vite, then serve with nginx
# =============================================================================

# --- Builder ---
FROM node:24-alpine AS builder

WORKDIR /app

# Copy manifest first for layer caching
COPY package.json ./

# npm install with workarounds for overlay2 bug:
# - Set cache to /tmp to avoid overlay2 issues
# - --no-optional: reduce package count
# - --no-audit --no-fund: skip network checks
# - --no-package-lock: avoid lockfile format issues
ENV NPM_CONFIG_CACHE=/tmp/.npm
RUN npm install --no-optional --no-audit --no-fund --no-package-lock --loglevel=error

COPY . .
RUN npm run build

# --- Runtime ---
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx-admin.conf /etc/nginx/conf.d/default.conf

EXPOSE 5174
CMD ["nginx", "-g", "daemon off;"]
