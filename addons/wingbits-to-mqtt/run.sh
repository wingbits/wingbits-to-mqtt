#!/usr/bin/with-contenv bashio

# Generate config.yaml from add-on options
cat > /app/wingbits-config.yaml << EOF
prometheus:
  sources:
$(for i in $(bashio::config 'prometheus_sources'); do
  echo "    - url: \"$(bashio::config "prometheus_sources[${i}].url")\""
  echo "      label: \"$(bashio::config "prometheus_sources[${i}].label")\""
done)

mqtt:
  broker: "$(bashio::config 'mqtt.broker')"
  client_id: "$(bashio::config 'mqtt.client_id')"
  topic_base: "$(bashio::config 'mqtt.topic_base')"
  username: "$(bashio::config 'mqtt.username')"
  password: "$(bashio::config 'mqtt.password')"

fetch_interval_seconds: $(bashio::config 'fetch_interval_seconds')
EOF

# Start the application
cd /app
exec ./wingbits-to-mqtt 