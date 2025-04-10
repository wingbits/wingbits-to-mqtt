ARG BUILD_FROM
FROM $BUILD_FROM

# Set shell
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Install required packages
RUN apk add --no-cache \
    go=~1.21 \
    git=~2.43 \
    build-base=~0.5 \
    && go version

# Set working directory
WORKDIR /app

# Copy the application
COPY . .

# Build the application
RUN go build -o wingbits-to-mqtt

# Copy the run script
COPY run.sh /
RUN chmod a+x /run.sh

# Set the entrypoint
ENTRYPOINT [ "/run.sh" ] 