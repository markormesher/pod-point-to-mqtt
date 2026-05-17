package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/markormesher/pod-point-to-mqtt/internal/settings"
)

type MQTTClientWrapper struct {
	client      mqtt.Client
	topicPrefix string
}

func setupMQTTClient(s settings.Settings) (*MQTTClientWrapper, error) {
	slog.Info("connecting to MQTT server...")
	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(s.MQTTConnectionString)

	mqttClient := mqtt.NewClient(mqttOpts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &MQTTClientWrapper{
		client:      mqttClient,
		topicPrefix: s.MQTTTopicPrefix,
	}, nil
}

func (w *MQTTClientWrapper) publish(topic string, payload any) {
	if w.client == nil || !w.client.IsConnected() {
		slog.Error("publish() called but MQTT client is not set up or is not connected")
		os.Exit(1)
	}

	var realPayload string
	switch payload := payload.(type) {
	case string:
		realPayload = payload

	default:
		jsonString, err := json.Marshal(payload)
		if err != nil {
			slog.Error("error marshalling MQTT payload", "error", err)
			os.Exit(1)
		}
		realPayload = string(jsonString)
	}

	topic = fmt.Sprintf("%s/%s", w.topicPrefix, topic)
	slog.Debug("publishing message", "topic", topic, "payload", realPayload)
	if token := w.client.Publish(topic, 0, false, realPayload); token.Wait() && token.Error() != nil {
		slog.Error("error publishing MQTT message", "error", token.Error())
		os.Exit(1)
	}
}
