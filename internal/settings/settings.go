package settings

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Settings struct {
	MqttConnectionString string
	MqttTopicPrefix      string
	UpdateInterval       int
	PodPointUsername     string
	PodPointPassword     string
	DataDir              string
}

func GetSettings() (Settings, error) {
	mqttConnectionString := os.Getenv("MQTT_CONNECTION_STRING")
	if len(mqttConnectionString) == 0 {
		mqttConnectionString = "tcp://0.0.0.0:1883"
	}

	mqttTopicPrefix := strings.TrimRight(os.Getenv("MQTT_TOPIC_PREFIX"), "/")
	if len(mqttTopicPrefix) == 0 {
		mqttTopicPrefix = "pod_point"
	}

	updateIntervalStr := os.Getenv("UPDATE_INTERVAL")
	if len(updateIntervalStr) == 0 {
		updateIntervalStr = "0"
	}
	updateInterval, err := strconv.Atoi(updateIntervalStr)
	if err != nil {
		return Settings{}, fmt.Errorf("could not parse update interval as an integer: %w", err)
	}

	podPointUsername := os.Getenv("POD_POINT_USERNAME")
	podPointPassword := os.Getenv("POD_POINT_PASSWORD")

	dataDir := os.Getenv("DATA_DIR")
	dataDirStat, err := os.Stat(dataDir)
	if err != nil {
		return Settings{}, fmt.Errorf("could not stat data directory: %w", err)
	}

	if !dataDirStat.IsDir() {
		return Settings{}, fmt.Errorf("data dir '%s' is not a directory", dataDir)
	}

	return Settings{
		MqttConnectionString: mqttConnectionString,
		MqttTopicPrefix:      mqttTopicPrefix,
		UpdateInterval:       updateInterval,
		PodPointUsername:     podPointUsername,
		PodPointPassword:     podPointPassword,
		DataDir:              dataDir,
	}, nil
}
