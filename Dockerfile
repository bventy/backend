# Step 1: Build Stage
FROM golang:1.24-alpine AS builder

# Install build dependencies for CGO (required by webp/imaging)
RUN apk add --no-cache gcc musl-dev libwebp-dev

WORKDIR /app

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o api cmd/api/main.go

# Step 2: Final Stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates libwebp-dev tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/api .

# Expose port
EXPOSE 8082

# Run the app
CMD ["./api"]
