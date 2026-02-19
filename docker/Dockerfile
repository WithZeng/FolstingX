# syntax=docker/dockerfile:1.7

FROM golang:1.22 AS backend-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum* ./backend/
RUN cd backend && go mod download || true
COPY backend ./backend
RUN cd backend && CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /out/folstingx ./cmd/server

FROM node:22 AS frontend-builder
WORKDIR /app
COPY frontend/package*.json ./frontend/
RUN cd frontend && npm ci
COPY frontend ./frontend
RUN cd frontend && npm run build

FROM debian:bookworm-slim
WORKDIR /opt/folstingx
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=backend-builder /out/folstingx /usr/local/bin/folstingx
COPY --from=frontend-builder /app/frontend/dist ./frontend-dist
COPY config/config.example.yaml ./config/config.yaml
EXPOSE 8080
CMD ["/usr/local/bin/folstingx"]
