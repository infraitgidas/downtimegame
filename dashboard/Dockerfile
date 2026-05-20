# =============================================================================
# Dashboard — Downtime Game
# Multi-stage: build React+Vite, then serve with nginx
# =============================================================================

# --- Builder ---
FROM node:24-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm install

COPY . .
RUN npm run build

# --- Runtime ---
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx-dashboard.conf /etc/nginx/conf.d/default.conf

EXPOSE 5173
CMD ["nginx", "-g", "daemon off;"]
