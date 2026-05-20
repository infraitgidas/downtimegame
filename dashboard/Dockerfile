# =============================================================================
# Dashboard — Downtime Game
# Multi-stage: build React+Vite, then serve with nginx
# =============================================================================

# --- Builder ---
FROM node:24-alpine AS builder

WORKDIR /app

COPY package.json ./

ENV NODE_OPTIONS="--use-system-ca"
ENV NPM_CONFIG_CACHE=/tmp/.npm
RUN --mount=type=cache,target=/tmp/.npm \
    apk add --no-cache ca-certificates && \
    update-ca-certificates && \
    npm install \
        --use-system-ca \
        --no-optional --no-audit --no-fund \
        --no-package-lock \
        --loglevel=error

COPY . .
RUN npm run build

# --- Runtime ---
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx-dashboard.conf /etc/nginx/conf.d/default.conf

EXPOSE 5173
CMD ["nginx", "-g", "daemon off;"]
