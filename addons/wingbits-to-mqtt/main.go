package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// Version information set during build
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

type Config struct {
	Prometheus struct {
		Sources []struct {
			URL   string `yaml:"url" json:"url"`
			Label string `yaml:"label" json:"label"`
		} `yaml:"sources"`
	} `yaml:"prometheus" json:"prometheus_sources"`
	MQTT struct {
		Broker    string `yaml:"broker" json:"broker"`
		ClientID  string `yaml:"client_id" json:"client_id"`
		TopicBase string `yaml:"topic_base" json:"topic_base"`
		Username  string `yaml:"username" json:"username"`
		Password  string `yaml:"password" json:"password"`
	} `yaml:"mqtt"`
	FetchIntervalSeconds int `yaml:"fetch_interval_seconds" json:"fetch_interval_seconds"`
}

type MetricInfo struct {
	Help string
	Type string
}

type HassDiscoveryConfig struct {
	Name              string     `json:"name"`
	StateTopic        string     `json:"state_topic"`
	UniqueID          string     `json:"unique_id"`
	Device            HassDevice `json:"device"`
	UnitOfMeasurement string     `json:"unit_of_measurement,omitempty"`
	Icon              string     `json:"icon,omitempty"`
}

type HassDevice struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
}

type MetricProcessor struct {
	config           *Config
	publishedConfigs map[string]bool
	helpRegex        *regexp.Regexp
	typeRegex        *regexp.Regexp
	mqttClient       mqtt.Client
}

func loadConfig(configPath string) (*Config, error) {
	// First try to read from Home Assistant options.json
	optionsPath := "/data/options.json"
	if _, err := os.Stat(optionsPath); err == nil {
		data, err := os.ReadFile(optionsPath)
		if err != nil {
			return nil, fmt.Errorf("error reading options.json: %w", err)
		}
		fmt.Println("options.json found")
		fmt.Printf("options.json content: %s", string(data))
		var config Config
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("error parsing options.json: %w", err)
		}
		fmt.Printf("options.json parsed: %+v", config)
		// Set default fetch interval if not specified
		if config.FetchIntervalSeconds <= 0 {
			config.FetchIntervalSeconds = 60
		}

		if err := validateConfig(&config); err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}

		return &config, nil
	}

	// Fall back to YAML config if options.json doesn't exist
	if configPath == "" {
		configPath = "wingbits-config.yaml"
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

func validateConfig(config *Config) error {

	if len(config.Prometheus.Sources) == 0 {
		return fmt.Errorf("no prometheus sources configured")
	}

	for i, source := range config.Prometheus.Sources {
		if source.URL == "" {
			return fmt.Errorf("prometheus source %d: url is required", i)
		}
		if source.Label == "" {
			return fmt.Errorf("prometheus source %d: label is required", i)
		}
	}

	// Check MQTT settings
	if config.MQTT.Broker == "" {
		return fmt.Errorf("MQTT broker is required")
	}
	if config.MQTT.ClientID == "" {
		return fmt.Errorf("MQTT client ID is required")
	}
	if config.MQTT.TopicBase == "" {
		return fmt.Errorf("MQTT topic base is required")
	}

	// Check fetch interval
	if config.FetchIntervalSeconds <= 0 {
		return fmt.Errorf("fetch interval must be positive")
	}

	return nil
}

func NewMetricProcessor(config *Config) (*MetricProcessor, error) {
	mqttClient, err := connectMQTT(config.MQTT.Broker, config.MQTT.ClientID, config.MQTT.Username, config.MQTT.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}

	return &MetricProcessor{
		config:           config,
		publishedConfigs: make(map[string]bool),
		helpRegex:        regexp.MustCompile(`^#\s+HELP\s+(\w+)\s+(.*)$`),
		typeRegex:        regexp.MustCompile(`^#\s+TYPE\s+(\w+)\s+(\w+)$`),
		mqttClient:       mqttClient,
	}, nil
}

func (mp *MetricProcessor) Close() {
	if mp.mqttClient != nil && mp.mqttClient.IsConnected() {
		mp.mqttClient.Disconnect(250)
	}
}

func (mp *MetricProcessor) Run() {
	defer mp.Close()

	ticker := time.NewTicker(time.Duration(mp.config.FetchIntervalSeconds) * time.Second)
	defer ticker.Stop()

	// Initial run
	mp.processAllSources()

	// Periodic runs
	for range ticker.C {
		mp.processAllSources()
	}
}

func (mp *MetricProcessor) processAllSources() {
	for _, source := range mp.config.Prometheus.Sources {
		mp.processMetrics(source.URL, source.Label)
	}
}

func (mp *MetricProcessor) processMetrics(prometheusURL, sourceLabel string) {
	log.Printf("Processing metrics from %s (%s)", prometheusURL, sourceLabel)

	metricsData, err := mp.fetchPrometheusMetrics(prometheusURL)
	if err != nil {
		log.Printf("Error fetching metrics: %v", err)
		return
	}

	metricMetadata := mp.collectMetadata(metricsData)
	mp.publishMetrics(metricsData, metricMetadata, sourceLabel)
}

func (mp *MetricProcessor) fetchPrometheusMetrics(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch metrics: status code %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(bodyBytes), nil
}

func (mp *MetricProcessor) collectMetadata(metricsData string) map[string]MetricInfo {
	metricMetadata := make(map[string]MetricInfo)

	for _, line := range strings.Split(metricsData, "\n") {
		line = strings.TrimSpace(line)

		if helpMatch := mp.helpRegex.FindStringSubmatch(line); len(helpMatch) == 3 {
			metricName := helpMatch[1]
			helpText := helpMatch[2]
			info := metricMetadata[metricName]
			info.Help = helpText
			metricMetadata[metricName] = info
		} else if typeMatch := mp.typeRegex.FindStringSubmatch(line); len(typeMatch) == 3 {
			metricName := typeMatch[1]
			metricType := typeMatch[2]
			info := metricMetadata[metricName]
			info.Type = metricType
			metricMetadata[metricName] = info
		}
	}

	return metricMetadata
}

func (mp *MetricProcessor) publishMetrics(metricsData string, metricMetadata map[string]MetricInfo, sourceLabel string) {
	lines := strings.Split(metricsData, "\n")
	publishedStateCount := 0
	skippedLineCount := 0
	configAttemptCount := 0
	configSuccessCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			skippedLineCount++
			continue
		}

		metricName, value, err := mp.parseMetricLine(line)
		if err != nil {
			log.Printf("Error parsing metric line: %v", err)
			skippedLineCount++
			continue
		}

		objectID := mp.sanitizeMetricName(metricName)
		labeledObjectID := fmt.Sprintf("%s_%s", sourceLabel, objectID)

		if !mp.publishedConfigs[labeledObjectID] {
			configAttemptCount++
			if info, ok := metricMetadata[metricName]; ok {
				if err := mp.publishDiscoveryConfig(metricName, info.Help, info.Type, sourceLabel); err != nil {
					log.Printf("Failed to publish discovery config for %s: %v", metricName, err)
				} else {
					mp.publishedConfigs[labeledObjectID] = true
					configSuccessCount++
				}
			}
		}

		stateTopic := fmt.Sprintf("%s%s/state", mp.config.MQTT.TopicBase, labeledObjectID)
		token := mp.mqttClient.Publish(stateTopic, 0, false, value)

		if token.Error() != nil {
			log.Printf("Failed to publish state for %s: %v", metricName, token.Error())
		} else {
			publishedStateCount++
		}
	}

	log.Printf("Published %d states, %d/%d configs, skipped %d lines",
		publishedStateCount, configSuccessCount, configAttemptCount, skippedLineCount)
}

func (mp *MetricProcessor) parseMetricLine(line string) (metricName, value string, err error) {
	labelStartIndex := strings.Index(line, "{")
	firstSpaceIndex := strings.Index(line, " ")
	lastSpaceIndex := strings.LastIndex(line, " ")

	if lastSpaceIndex == -1 {
		return "", "", fmt.Errorf("no value found in line")
	}

	value = line[lastSpaceIndex+1:]

	if labelStartIndex != -1 && (firstSpaceIndex == -1 || labelStartIndex < firstSpaceIndex) {
		metricName = line[:labelStartIndex]
	} else if firstSpaceIndex != -1 {
		metricName = line[:firstSpaceIndex]
	} else {
		return "", "", fmt.Errorf("no metric name found in line")
	}

	return metricName, value, nil
}

func (mp *MetricProcessor) sanitizeMetricName(name string) string {
	result := strings.ReplaceAll(name, ".", "_")
	return strings.ReplaceAll(result, "-", "_")
}

func (mp *MetricProcessor) publishDiscoveryConfig(metricName, helpText, metricType, sourceLabel string) error {
	objectID := mp.sanitizeMetricName(metricName)
	labeledObjectID := fmt.Sprintf("%s_%s", sourceLabel, objectID)

	configTopic := fmt.Sprintf("%s%s/config", mp.config.MQTT.TopicBase, labeledObjectID)
	stateTopic := fmt.Sprintf("%s%s/state", mp.config.MQTT.TopicBase, labeledObjectID)

	friendlyName := strings.ReplaceAll(strings.TrimPrefix(metricName, "wingbits_"), "_", " ")
	friendlyName = cases.Title(language.English).String(friendlyName)
	if len(friendlyName) == 0 {
		friendlyName = objectID
	}

	icon := mp.determineIcon(metricName, metricType)

	configPayload := HassDiscoveryConfig{
		Name:              fmt.Sprintf("Wingbits %s %s", cases.Title(language.English).String(sourceLabel), friendlyName),
		StateTopic:        stateTopic,
		UniqueID:          fmt.Sprintf("wingbits_%s_%s", sourceLabel, objectID),
		UnitOfMeasurement: "",
		Icon:              icon,
		Device: HassDevice{
			Identifiers:  []string{fmt.Sprintf("%s_%s", mp.config.MQTT.ClientID, sourceLabel)},
			Name:         fmt.Sprintf("Wingbits %s", cases.Title(language.English).String(sourceLabel)),
			Manufacturer: "Wingbits",
			Model:        "Prometheus Forwarder",
		},
	}

	jsonPayload, err := json.Marshal(configPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal config JSON: %w", err)
	}

	token := mp.mqttClient.Publish(configTopic, 1, true, jsonPayload)
	if token.WaitTimeout(2*time.Second) && token.Error() != nil {
		return fmt.Errorf("failed to publish config: %w", token.Error())
	}

	return nil
}

func (mp *MetricProcessor) determineIcon(metricName, metricType string) string {
	if strings.Contains(metricName, "connection") {
		return "mdi:connection"
	} else if strings.Contains(metricName, "total") || metricType == "counter" {
		return "mdi:counter"
	} else if strings.Contains(metricName, "version") {
		return "mdi:tag"
	}
	return "mdi:gauge"
}

func connectMQTT(broker, clientID, username, password string) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Minute)

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	})
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Println("Connected to MQTT broker")
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	return client, nil
}

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file (default: config.yaml)")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("wingbits-to-mqtt version %s (commit: %s, built: %s)\n", version, commit, date)
		return
	}

	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and run the metric processor
	processor, err := NewMetricProcessor(config)
	if err != nil {
		log.Fatalf("Failed to initialize metric processor: %v", err)
	}

	processor.Run()
}
