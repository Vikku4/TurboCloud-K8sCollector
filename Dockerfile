# ---------- Stage 1: Build ----------
FROM golang:1.22 AS build

WORKDIR /app

# Pre-copy go.mod/go.sum to leverage build cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build static Linux binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o k8s-collector ./cmd/k8s-collector

# ---------- Stage 2: Runtime ----------
# Distroless is nice & small; use alpine if you prefer a shell.
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy binary from builder
COPY --from=build /app/k8s-collector /app/k8s-collector

# The collector listens on 8080 by default
EXPOSE 8080

# Env vars are injected at runtime (.env equivalent)
#   PORT                (optional, default 8080)
#   DB_URL              (postgres connection string)
#   LOG_LEVEL           (info/debug/warn/error)
#   JWT_SECRET / etc.   (if/when you wire auth)
#
# ENTRYPOINT just runs the binary
ENTRYPOINT ["/app/k8s-collector"]
