package main

import (
	"fmt"
	"os"
	"time"

	"github.com/markormesher/pod-point-to-mqtt/internal/api"
	"github.com/markormesher/pod-point-to-mqtt/internal/logging"
	"github.com/markormesher/pod-point-to-mqtt/internal/settings"
)

var l = logging.Logger

func main() {
	s, err := settings.GetSettings()
	if err != nil {
		l.Error("Error getting settings", "error", err)
		os.Exit(1)
	}

	mqttClient, err := setupMqttClient(s)
	if err != nil {
		l.Error("Error setting up MQTT client", "error", err)
		os.Exit(1)
	}

	ppApi, err := api.NewApi(s)
	if err != nil {
		l.Error("Error setting up Pod-Point API", "error", err)
		os.Exit(1)
	}

	if s.UpdateInterval <= 0 {
		l.Info("Running once then exiting because update interval is <= 0")
		doUpdate(&ppApi, mqttClient)
	} else {
		l.Info("Running forever", "interval", s.UpdateInterval)
		for {
			doUpdate(&ppApi, mqttClient)
			time.Sleep(time.Duration(s.UpdateInterval * int(time.Second)))
		}
	}
}

func doUpdate(ppApi *api.PodPointApi, mqttClient *MqttClientWrapper) {
	now := time.Now()

	pods, err := ppApi.GetPods()
	if err != nil {
		l.Error("Error getting pods", "error", err)
		return
	}

	for _, pod := range pods {
		// map statuses to their door id for lookup later
		doorStatus := map[int]api.PodStatus{}
		for _, status := range pod.Statuses {
			doorStatus[status.DoorID] = status
		}

		prefix := fmt.Sprintf("pods/%d/state", pod.ID)

		// pod details
		mqttClient.publish(fmt.Sprintf("%s/id", prefix), pod.ID)
		mqttClient.publish(fmt.Sprintf("%s/pod_point_id", prefix), pod.PodPointID)
		mqttClient.publish(fmt.Sprintf("%s/name", prefix), pod.Name)
		mqttClient.publish(fmt.Sprintf("%s/description", prefix), pod.Description)
		mqttClient.publish(fmt.Sprintf("%s/last_contact", prefix), pod.LastContactTime)

		// model details
		mqttClient.publish(fmt.Sprintf("%s/model_id", prefix), pod.Model.ID)
		mqttClient.publish(fmt.Sprintf("%s/model_name", prefix), pod.Model.Name)
		mqttClient.publish(fmt.Sprintf("%s/model_vendor", prefix), pod.Model.Vendor)
		mqttClient.publish(fmt.Sprintf("%s/model_image_url", prefix), pod.Model.ImageUrl)

		// connector details
		for _, connector := range pod.Connectors {
			c := connector.Connector
			connectorPrefix := fmt.Sprintf("%s/connectors/%d", prefix, c.ID)
			mqttClient.publish(fmt.Sprintf("%s/id", connectorPrefix), c.ID)
			mqttClient.publish(fmt.Sprintf("%s/door", connectorPrefix), c.Door)
			mqttClient.publish(fmt.Sprintf("%s/door_id", connectorPrefix), c.DoorID)
			mqttClient.publish(fmt.Sprintf("%s/power", connectorPrefix), c.Power)
			mqttClient.publish(fmt.Sprintf("%s/current", connectorPrefix), c.Current)
			mqttClient.publish(fmt.Sprintf("%s/voltage", connectorPrefix), c.Voltage)
			mqttClient.publish(fmt.Sprintf("%s/charging_method", connectorPrefix), c.ChargeMethod)
			mqttClient.publish(fmt.Sprintf("%s/has_cable", connectorPrefix), c.HasCable)
			mqttClient.publish(fmt.Sprintf("%s/socket_type", connectorPrefix), c.Socket.Type)
			mqttClient.publish(fmt.Sprintf("%s/socket_description", connectorPrefix), c.Socket.Description)
			mqttClient.publish(fmt.Sprintf("%s/socket_ocpp_name", connectorPrefix), c.Socket.OCPPName)
			mqttClient.publish(fmt.Sprintf("%s/socket_ocpp_code", connectorPrefix), c.Socket.OCPPCode)

			status, ok := doorStatus[c.DoorID]
			if !ok {
				l.Warn("No status found for door", "doorID", c.DoorID)
				continue
			}

			mqttClient.publish(fmt.Sprintf("%s/status_name", connectorPrefix), status.Name)
			mqttClient.publish(fmt.Sprintf("%s/status_key", connectorPrefix), status.KeyName)
			mqttClient.publish(fmt.Sprintf("%s/status_label", connectorPrefix), status.Label)
		}

		// schedule details
		for _, schedule := range pod.ChargeSchedules {
			schedulePrefix := fmt.Sprintf("%s/charging_schedules/%d", prefix, schedule.Day)
			mqttClient.publish(fmt.Sprintf("%s/start_time", schedulePrefix), schedule.StartTime)
			mqttClient.publish(fmt.Sprintf("%s/end_time", schedulePrefix), schedule.EndTime)
			mqttClient.publish(fmt.Sprintf("%s/active", schedulePrefix), schedule.Status.Active)
		}

		if pod.ChargeOveride != nil {
			mqttClient.publish(fmt.Sprintf("%s/charging_override_exists", prefix), true)
			mqttClient.publish(fmt.Sprintf("%s/charging_override_ends_at", prefix), pod.ChargeOveride.EndsAt)
		} else {
			mqttClient.publish(fmt.Sprintf("%s/charging_override_exists", prefix), false)
			mqttClient.publish(fmt.Sprintf("%s/charging_override_ends_at", prefix), "")
		}

		// charging mode logic:
		// - if there is no override => schedule mode
		// - if there is an override...
		//     - if it has no end date => manual mode
		//     - if it has an end date in the future => override mode
		//     - if it has an end date in the past => schedule mode
		chargeMode := ""
		if pod.ChargeOveride == nil {
			chargeMode = "SCHEDULE"
		} else {
			if pod.ChargeOveride.EndsAt.IsZero() {
				chargeMode = "MANUAL"
			} else if pod.ChargeOveride.EndsAt.After(now) {
				chargeMode = "OVERRIDE"
			} else {
				chargeMode = "SCHEDULE"
			}
		}
		mqttClient.publish(fmt.Sprintf("%s/charging_mode", prefix), chargeMode)

		// schedule logic:
		// - if the schedule for today IS NOT active, charging is allowed all day
		// - if the schedule for today IS active, charging is allowed between the start and end time
		chargingAllowedBySchedule := false
		timeStr := now.Format("15:04")
		dateInt := int(now.Weekday())
		if dateInt == 0 {
			dateInt = 7
		}
		for _, schedule := range pod.ChargeSchedules {
			if schedule.Day != dateInt {
				continue
			}

			if !schedule.Status.Active {
				chargingAllowedBySchedule = true
				continue
			}

			chargingAllowedBySchedule = schedule.StartTime <= timeStr && schedule.EndTime >= timeStr
		}
		mqttClient.publish(fmt.Sprintf("%s/charging_allowed_by_schedule", prefix), chargingAllowedBySchedule)

		// overall, is charging allowed right now?
		chargingAllowed := false
		if chargeMode == "SCHEDULE" {
			chargingAllowed = chargingAllowedBySchedule
		} else {
			chargingAllowed = true
		}
		mqttClient.publish(fmt.Sprintf("%s/charging_allowed", prefix), chargingAllowed)
	}

	mqttClient.publish("_meta/last_seen", now.Format(time.RFC3339))
}
