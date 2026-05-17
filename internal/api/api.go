package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
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
	client         *resty.Client
	userID         int
	apiToken       string
	apiTokenExpiry time.Time
	refreshToken   string
}

func NewAPI(s settings.Settings) (PodPointAPI, error) {
	api := PodPointAPI{
		s:      s,
		client: resty.New(),
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

	req, err := api.getAuthedReqeust()
	if err != nil {
		return fmt.Errorf("error getting user ID: %w", err)
	}

	url := fmt.Sprintf("%s/auth", podpointAPIBaseURL)
	res, err := req.Get(url)
	if err != nil {
		return fmt.Errorf("error getting user ID: %w", err)
	}

	if res.StatusCode() != http.StatusOK {
		return fmt.Errorf("error getting user ID: status %d", res.StatusCode())
	}

	type accountResponse struct {
		User struct {
			ID int `json:"id"`
		} `json:"users"`
	}

	var resParsed accountResponse
	err = json.Unmarshal(res.Body(), &resParsed)
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

	req, err := api.getAuthedReqeust()
	if err != nil {
		return nil, fmt.Errorf("error getting pods: %w", err)
	}

	url := fmt.Sprintf("%s/users/%d/pods?perpage=all&include=statuses,model,unit_connectors,charge_schedules,charge_override", podpointAPIBaseURL, api.userID)
	res, err := req.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error getting pods: %w", err)
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("error getting pods: status %d", res.StatusCode())
	}

	type podsResponse struct {
		Pods []Pod `json:"pods"`
	}

	body := res.Body()

	var resParsed podsResponse
	err = json.Unmarshal(body, &resParsed)
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

func (api *PodPointAPI) getPlainReqeust() *resty.Request {
	req := api.client.NewRequest()
	req.SetHeader("content-type", "application/json; charset=UTF-8")

	return req
}

func (api *PodPointAPI) getAuthedReqeust() (*resty.Request, error) {
	err := api.loadAuthToken()

	if err != nil {
		return nil, err
	}

	req := api.getPlainReqeust()
	req.SetHeader("authorization", fmt.Sprintf("Bearer %s", api.apiToken))

	return req, nil
}
