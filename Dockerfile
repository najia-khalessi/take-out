# Stage 1: Build the Go application
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

# Copy the source code
COPY . .

# Build the application
# -ldflags="-w -s" reduces the size of the binary
# CGO_ENABLED=0 prevents the usage of C libraries
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o takeout-backend main.go

# Stage 2: Create the final, lightweight image
FROM alpine:latest

WORKDIR /root/

# Copy the pre-built binary from the builder stage
COPY --from=builder /app/takeout-backend .
# Copy the config file
COPY config.env .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./takeout-backend"]
