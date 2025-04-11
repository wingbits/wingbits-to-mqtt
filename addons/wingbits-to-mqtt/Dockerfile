# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o wingbits-to-mqtt

# Final stage
FROM homeassistant/amd64-base:latest

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/wingbits-to-mqtt .
# Copy config files
COPY run.sh .
RUN chmod +x run.sh

# Run the application
CMD ["/app/run.sh"]
