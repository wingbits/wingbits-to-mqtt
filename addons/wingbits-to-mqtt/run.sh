#!/usr/bin/with-contenv bashio

bashio::log.info "Starting wingbits-to-mqtt application..."

# Start the application
cd /app
exec ./wingbits-to-mqtt