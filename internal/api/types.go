package api

import "time"

type Pod struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	PodPointID      string `json:"ppid"`
	IsPAYGCharge    bool   `json:"payg"`
	IsHomeCharger   bool   `json:"home"`
	IsPublicCharger bool   `json:"public"`
	IsEVZoneCharger bool   `json:"evZone"`
	Location        struct {
		Lat float32 `json:"lat"`
		Lon float32 `json:"lng"`
	} `json:"location"`
	LastContactTimeStr string `json:"last_contact_at"`
	LastContactTime    time.Time

	Model           PodModel            `json:"model"`
	Statuses        []PodStatus         `json:"statuses"`
	Connectors      []PodConnector      `json:"unit_connectors"`
	ChargeSchedules []PodChargeSchedule `json:"charge_schedules"`
	ChargeOveride   *PodChargeOverride  `json:"charge_override"`
}

type PodModel struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Vendor   string `json:"vendor"`
	ImageUrl string `json:"image_url"`
}

type PodStatus struct {
	ID      int    `json:"id"`
	Door    string `json:"door"`
	DoorID  int    `json:"door_id"`
	Name    string `json:"name"`
	KeyName string `json:"key_name"`
	Label   string `json:"label"`
}

type PodConnector struct {
	// these are nested an extra layer on the API response for some reason
	Connector struct {
		ID           int     `json:"id"`
		Door         string  `json:"door"`
		DoorID       int     `json:"door_id"`
		Power        float32 `json:"power"`
		Current      float32 `json:"current"`
		Voltage      float32 `json:"voltage"`
		ChargeMethod string  `json:"charge_method"`
		HasCable     bool    `json:"has_cable"`
		Socket       struct {
			Type        string `json:"type"`
			Description string `json:"description"`
			OCPPName    string `json:"ocpp_name"`
			OCPPCode    int    `json:"ocpp_code"`
		} `json:"socket"`
	} `json:"connector"`
}

type PodChargeSchedule struct {
	// note: all schedules start and end on the same day, so we ignore the end-day value
	ID        string `json:"uid"`
	Day       int    `json:"start_day"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Status    struct {
		Active bool `json:"is_active"`
	} `json:"status"`
}

type PodChargeOverride struct {
	PodPointID string `json:"ppid"`
	EndsAtStr  string `json:"ends_at"`
	EndsAt     time.Time
}
