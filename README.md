# Wingbits to MQTT Forwarder

This application fetches Metrics from one or more wingbits clients, and sends them to a mqtt server. Adopted for use with homeassistant.

## Configuration

The application is configured using a YAML file. By default, it looks for `config.yaml` in the current directory, but you can specify a different path as a command-line argument.

### Example Configuration

```yaml
prometheus:
  sources:
    - url: "http://192.168.1.10:8088/metrics"
      label: "device1"
    - url: "http://192.168.1.11:8088/metrics"
      label: "device2"

mqtt:
  broker: "tcp://192.168.1.3:1883"
  client_id: "wingbits-mqtt"
  topic_base: "homeassistant/sensor/wingbits/"
  username: "mqttusername"
  password: "mqttpassword"

fetch_interval_seconds: 60
```

### Configuration Options

#### Prometheus Sources
- `url`: The URL of the Prometheus metrics endpoint
- `label`: A unique label for this source (used in MQTT topics and entity names)

#### MQTT Settings
- `broker`: The MQTT broker address
- `client_id`: The MQTT client ID
- `topic_base`: The base topic for Home Assistant MQTT Discovery
- `username`: MQTT username
- `password`: MQTT password

#### Other Settings
- `fetch_interval_seconds`: How often to fetch metrics (in seconds)

## Usage

Run the application with:

```bash
# Use default config.yaml
./wingbits-to-mqtt

# Or specify a custom config file
./wingbits-to-mqtt -config /path/to/config.yaml

# Show version information
./wingbits-to-mqtt -version
```

## Building

### Local Build

```bash
go build -o wingbits-to-mqtt
```

### Cross-Compilation

```bash
# For Linux ARM
GOOS=linux GOARCH=arm GOARM=6 go build -o wingbits-to-mqtt-linux-arm

# For Windows
GOOS=windows GOARCH=amd64 go build -o wingbits-to-mqtt.exe
```

### Using GoReleaser

This project uses [GoReleaser](https://goreleaser.com/) for building and releasing:

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Build for all platforms
goreleaser build --snapshot --rm-dist

# Create a release
goreleaser release --snapshot --rm-dist
```

## Home Assistant Integration

The application automatically creates MQTT entities in Home Assistant using the MQTT Discovery feature. Each Prometheus source will have its own device, and metrics will be grouped under these devices.

Entities will be named according to the pattern: `Wingbits [Source Label] [Metric Name]` 