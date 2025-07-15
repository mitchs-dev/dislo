# Variables
ARG GOLANG_VERSION=1.23
ARG DEBIAN_VERSION=11

# Build stage
FROM golang:${GOLANG_VERSION}-alpine AS builder
WORKDIR /opt/dislo

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o ./bin ./cmd/main.go
RUN chmod +x ./bin

# Final stage
FROM gcr.io/distroless/static-debian${DEBIAN_VERSION}
WORKDIR /opt/dislo/
COPY --from=builder /opt/dislo/bin /opt/dislo/bin
USER nonroot:nonroot

# Labels
LABEL maintainer="Mitchell Stanton <email@mitchs.dev>"
LABEL AUTHOR="Mitchell Stanton  <email@mitchs.dev>"

ENTRYPOINT ["/opt/dislo/bin"]