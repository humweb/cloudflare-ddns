# Stage 1: Build the frontend
FROM golang:1.23.2 AS backend-build

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
ENV CGO_ENABLED=0

# Download dependencies
RUN go mod download

# Copy the backend code (now in root)
COPY . .


# Build the Go application
RUN go build -ldflags="-s -w" -o /app/main /app

# Stage 2: Final stage
FROM scratch

WORKDIR /app

# Copy the built backend binary
COPY --from=backend-build /app/main /app/main

# Import from builder
COPY --from=backend-build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=backend-build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=backend-build /etc/passwd /etc/passwd
COPY --from=backend-build /etc/group /etc/group

# Command to run the executable
CMD ["/app/main", "--repeat"]
