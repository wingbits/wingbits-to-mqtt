#!/usr/bin/with-contenv bashio

# Debug: Check if options.json exists and its contents
echo "Checking /data/options.json..."
ls -l /data/options.json || echo "options.json not found"
echo "Contents of /data/options.json:"
cat /data/options.json || echo "Could not read options.json"

bashio::log.info "Generating configuration for wingbits-to-mqtt..."

# Read config values into variables first for clarity and debugging
declare mqtt_broker
declare mqtt_client_id
declare mqtt_topic_base
declare mqtt_username
declare mqtt_password
declare fetch_interval

# Use bashio::config to read values, providing default empty string for optional password
mqtt_broker=$(bashio::config 'mqtt.broker')
mqtt_client_id=$(bashio::config 'mqtt.client_id')
mqtt_topic_base=$(bashio::config 'mqtt.topic_base')
mqtt_username=$(bashio::config 'mqtt.username')
mqtt_password=$(bashio::config 'mqtt.password' '') # Default to empty string if null/missing
fetch_interval=$(bashio::config 'fetch_interval_seconds')

# Log fetched values (adjust log level as needed, avoid logging passwords in production)
bashio::log.info "MQTT Broker: ${mqtt_broker}"
bashio::log.info "MQTT Client ID: ${mqtt_client_id}"
bashio::log.info "MQTT Topic Base: ${mqtt_topic_base}"
bashio::log.info "MQTT Username: ${mqtt_username}"
# bashio::log.info "MQTT Password: [REDACTED]" # Avoid logging password directly
bashio::log.info "Fetch Interval Raw: ${fetch_interval}"

# Validate fetch_interval - check if it's a positive integer
if ! [[ "${fetch_interval}" =~ ^[0-9]+$ ]]; then
    bashio::log.warning "Fetch interval ('${fetch_interval}') is not a valid number. Check add-on configuration. Using default 60."
    fetch_interval=60
else
    bashio::log.debug "Fetch Interval Validated: ${fetch_interval}"
fi

# Generate config file
cat > /app/wingbits-config.yaml << EOF
prometheus:
  sources:
$(
  # Check if prometheus_sources is defined and is a list
  if bashio::config.is_list 'prometheus_sources'; then
    # Use bashio's length filter to get the count
    count=$(bashio::config 'prometheus_sources | length')
    bashio::log.debug "Found ${count} Prometheus sources."
    # Loop from 0 to count-1
    for i in $(seq 0 $((count - 1))); do
      # Fetch URL and Label for the current index
      url=$(bashio::config "prometheus_sources[${i}].url")
      label=$(bashio::config "prometheus_sources[${i}].label")
      bashio::log.debug "Source ${i}: URL='${url}', Label='${label}'"

      # Basic check if URL looks like a variable placeholder (like the error)
      if [[ "${url}" == "\${"* ]]; then
         bashio::log.warning "Prometheus source URL '${url}' at index ${i} looks like an unparsed variable. Check add-on configuration."
      fi
      # Output YAML list item, ensuring proper quoting
      echo "    - url: \"${url}\""
      echo "      label: \"${label}\""
    done
  else
    bashio::log.warning "No Prometheus sources configured or 'prometheus_sources' is not a list. Check add-on configuration."
    # Output nothing here, resulting in 'sources: null' or 'sources: []' depending on YAML parser
  fi
)

mqtt:
  broker: "${mqtt_broker}"
  client_id: "${mqtt_client_id}"
  topic_base: "${mqtt_topic_base}"
  username: "${mqtt_username}"
  # Ensure password is included and quoted, even if empty
  password: "${mqtt_password}"

# Output the validated integer value
fetch_interval_seconds: ${fetch_interval}
EOF

bashio::log.info "Configuration generated at /app/wingbits-config.yaml"
# Optional: Log the generated config for debugging (might expose password if not careful)
# bashio::log.debug "Generated config:\n$(cat /app/wingbits-config.yaml)"

# Start the application
cd /app
bashio::log.info "Starting wingbits-to-mqtt application..."

# Use exec to replace the shell process with our application
exec s6-setuidgid root ./wingbits-to-mqtt