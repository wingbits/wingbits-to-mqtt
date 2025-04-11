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
FROM homeassistant/amd64-base:3.16

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/wingbits-to-mqtt .
# Copy config files
COPY run.sh .

# Create s6 service directory
RUN mkdir -p /etc/services.d/wingbits-to-mqtt

# Create s6 service run script
RUN printf '#!/usr/bin/with-contenv bashio\ncd /app\nexec /app/run.sh\n' > /etc/services.d/wingbits-to-mqtt/run && \
    chmod +x /etc/services.d/wingbits-to-mqtt/run

# Make sure run.sh is executable
RUN chmod +x /app/run.sh
