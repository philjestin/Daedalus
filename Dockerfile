# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
ARG VITE_API_URL=""
ENV VITE_API_URL=$VITE_API_URL
RUN npm run build

# Stage 2: Build backend
FROM golang:1.24-alpine AS backend
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 3: Runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend /app/server .
COPY --from=frontend /app/web/dist ./web/dist
COPY migrations ./migrations
ENV PORT=8080
ENV UPLOAD_DIR=/app/uploads
EXPOSE 8080
CMD ["./server"]
