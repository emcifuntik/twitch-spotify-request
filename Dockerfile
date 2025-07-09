# Multi-stage build for frontend and backend

# Frontend build stage
FROM node:22-slim AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm i
COPY frontend/ ./
RUN npm run build

# Backend build stage
FROM golang:1.24-alpine AS backend-builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o twitch-spotify-request ./cmd/server/main.go

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates mariadb-client
WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /app/twitch-spotify-request .

# Copy frontend build
COPY --from=frontend-builder /app/frontend/dist ./web/static

# Create web directory structure
RUN mkdir -p ./web/static

CMD sh -c "until mariadb-admin ping -h \"$DB_HOST\" -P \"$DB_PORT\" --silent; do echo 'Waiting for MariaDB...'; sleep 1; done; ./twitch-spotify-request"
