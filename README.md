# Wingbits to MQTT Forwarder

This application fetches Metrics from one or more wingbits clients, and sends them to a mqtt server. Adopted for use with homeassistant.

## Installation

### Home Assistant Add-on (Recommended)

1. Add the Wingbits Add-on repository to your Home Assistant instance:
   - Go to **Settings** → **Add-ons** → **Add-on Store**
   - Click the three dots menu in the top right
   - Select **Repositories**
   - Add the repository URL: `https://github.com/wingbits/home-assistant-addons`

2. Install the "Wingbits to MQTT" add-on from the Add-on Store

3. Configure the add-on:
   - Set your Wingbits Prometheus sources (URLs and labels)
   - Configure MQTT settings (defaults to using Home Assistant's built-in MQTT broker)
   - Adjust the fetch interval if needed

4. Start the add-on

The add-on will automatically create MQTT entities in Home Assistant using the MQTT Discovery feature.

### Docker Installation

1. Clone this repository:
```bash
git clone https://github.com/wingbits/wingbits-to-mqtt.git
cd wingbits-to-mqtt
```

2. Create and edit your `config.yaml` file with your settings.

3. Make sure you have the Home Assistant network available:
```bash
docker network ls | grep homeassistant || docker network create homeassistant
```

4. Start the container:
```bash
docker-compose up -d
```

The container will automatically restart unless stopped manually.

### Manual Installation

Run the application with:

```bash
# Use default config.yaml
./wingbits-to-mqtt

# Or specify a custom config file
./wingbits-to-mqtt -config /path/to/config.yaml

# Show version information
./wingbits-to-mqtt -version
```

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

## Installation from GitHub Packages

You can install the latest release directly from GitHub Packages:

### Linux 
From https://github.com/wingbits/wingbits-to-mqtt/releases/latest

Example:
```bash
# Download the latest release
#Arm (raspberry pi)
curl -L -o wingbits-to-mqtt https://github.com/wingbits/wingbits-to-mqtt/releases/download/latest/wingbits-to-mqtt_Linux_arm.tar.gz

# Make it executable
chmod +x wingbits-to-mqtt

# Move to a directory in your PATH (optional)
sudo mv wingbits-to-mqtt /usr/local/bin/

curl -L -o config.yaml https://github.com/wingbits/wingbits-to-mqtt/blob/main/config.yaml
```
Edit the config.yaml to point at your wingbits stations and your homeassistant mqtt server.

## Home Assistant Integration

The application automatically creates MQTT entities in Home Assistant using the MQTT Discovery feature. Each Prometheus source will have its own device, and metrics will be grouped under these devices.

Entities will be named according to the pattern: `Wingbits [Source Label] [Metric Name]` 