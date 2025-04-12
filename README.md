# Wingbits to MQTT Forwarder

This application fetches Metrics from one or more wingbits clients, and sends them to a mqtt server. Adopted for use with homeassistant.

## Installation

### Home Assistant Add-on (Recommended)

1. Add the Wingbits Add-on repository to your Home Assistant instance:
   - Go to **Settings** → **Add-ons** → **Add-on Store**
   - Click the three dots menu in the top right
   - Select **Repositories**
   - Add the repository URL: `https://github.com/wingbits/wingbits-to-mqtt`

2. Install the "Wingbits to MQTT" add-on from the Add-on Store

3. Configure the add-on:
   - Set your Wingbits Prometheus sources (URLs and labels)
   - Configure MQTT settings (defaults to using Home Assistant's built-in MQTT broker)
   - Adjust the fetch interval if needed

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
