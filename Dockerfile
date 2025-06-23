# Stage 1: Build the Go application
FROM golang:1.22-alpine AS builder

# Set working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files and download dependencies
# This caches dependencies, so if they don't change, Docker can use a cached layer
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
# -o /app/gitlab-fork-cli: Specifies the output binary name and path within the container
# ./cmd: Specifies the directory containing your main package (e.g., cmd/main.go or cmd/fork.go)
# CGO_ENABLED=0: Disables CGO, which makes the binary statically linked and easier to run on minimal images
# GOOS=linux: Ensures the binary is built for Linux (the OS of our final image)
# GOARCH=amd64: Ensures the binary is built for AMD64 architecture
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/gitlab-fork-cli ./cmd

# Stage 2: Create a minimal runtime image
# Using alpine is a good balance between small size and having common utilities if needed.
# If you need an *extremely* minimal image and are sure your app has no OS dependencies, you could use FROM scratch
FROM alpine:latest

# Set working directory for the final application
WORKDIR /root/

# Copy the built binary from the builder stage
# --from=builder: Specifies to copy from the stage named 'builder'
# /app/gitlab-fork-cli: Path to the binary in the builder stage
# ./gitlab-fork-cli: Path to copy the binary to in the final image
COPY --from=builder /app/gitlab-fork-cli .

# Define the command to run the application
# This is the default command executed when the container starts
CMD ["./gitlab-fork-cli"]

