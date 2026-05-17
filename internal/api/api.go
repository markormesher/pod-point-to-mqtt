package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/markormesher/pod-point-to-mqtt/internal/settings"
)

var (
	podpointAPIBaseURL = "https://mobile-api.pod-point.com/api3/v5"
	googleAPIKey       = "AIzaSyCwhF8IOl_7qHXML0pOd5HmziYP46IZAGU"
	googleLoginURL     = "https://www.googleapis.com/identitytoolkit/v3/relyingparty/verifyPassword?key=" + googleAPIKey
	googleRefreshURL   = "https://securetoken.googleapis.com/v1/token?key=" + googleAPIKey
)

type PodPointAPI struct {
	s              settings.Settings
	client         *http.Client
	userID         int
	apiToken       string
	apiTokenExpiry time.Time
	refreshToken   string
}

func NewAPI(s settings.Settings) (PodPointAPI, error) {
	api := PodPointAPI{
		s: s,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	err := api.loadSavedAuthDetails()
	if err != nil {
		slog.Warn("error loading persisted auth details - continuing without them", "error", err)
	}

	err = api.loadAuthToken()
	if err != nil {
		return PodPointAPI{}, fmt.Errorf("error getting an auth token: %w", err)
	}

	err = api.loadUserID()
	if err != nil {
		return PodPointAPI{}, fmt.Errorf("error loading user ID: %w", err)
	}

	return api, nil
}

func (api *PodPointAPI) loadUserID() error {
	if api.userID != 0 {
		return nil
	}

	slog.Info("fetching user ID")

	url := fmt.Sprintf("%s/auth", podpointAPIBaseURL)
	req, err := api.authedReqeust("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error getting user ID: %w", err)
	}

	res, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("error getting user ID: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("error getting user ID: %s", res.Status)
	}

	type accountResponse struct {
		User struct {
			ID int `json:"id"`
		} `json:"users"`
	}

	var resParsed accountResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&resParsed)
	if err != nil {
		return fmt.Errorf("error parsing account response: %w", err)
	}

	api.userID = resParsed.User.ID

	return nil
}

func (api *PodPointAPI) GetPods() ([]Pod, error) {
	slog.Info("fetching pods")

	err := api.loadUserID()
	if err != nil {
		return nil, fmt.Errorf("error getting pods: %w", err)
	}

	url := fmt.Sprintf("%s/users/%d/pods?perpage=all&include=statuses,model,unit_connectors,charge_schedules,charge_override", podpointAPIBaseURL, api.userID)
	req, err := api.authedReqeust("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting pods: %w", err)
	}

	res, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting pods: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting pods: %s", res.Status)
	}

	type podsResponse struct {
		Pods []Pod `json:"pods"`
	}

	var resParsed podsResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&resParsed)
	if err != nil {
		return nil, fmt.Errorf("error parsing pods response: %w", err)
	}

	// parse timestamps
	for id, pod := range resParsed.Pods {
		pod.LastContactTime, err = time.Parse(time.RFC3339, pod.LastContactTimeStr)
		if err != nil {
			slog.Warn("error parsing contact timestamp", "raw", pod.LastContactTimeStr)
		}

		if pod.ChargeOveride != nil && pod.ChargeOveride.EndsAtStr != "" {
			pod.ChargeOveride.EndsAt, err = time.Parse(time.RFC3339, pod.ChargeOveride.EndsAtStr)
			if err != nil {
				slog.Warn("error parsing override timestamp", "raw", pod.ChargeOveride.EndsAtStr)
			}
		}

		resParsed.Pods[id] = pod
	}

	return resParsed.Pods, nil
}

func (api *PodPointAPI) unauthedReqeust(method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %w", err)
	}

	req.Header.Add("content-type", "application/json; charset=UTF-8")

	return req, nil
}

func (api *PodPointAPI) authedReqeust(method string, url string, body io.Reader) (*http.Request, error) {
	err := api.loadAuthToken()
	if err != nil {
		return nil, err
	}

	req, err := api.unauthedReqeust(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("authorization", fmt.Sprintf("Bearer %s", api.apiToken))

	return req, nil
}
