package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

type authPersistenceFile struct {
	Username     string `json:"username"`
	RefreshToken string `json:"refreshToken"`
}

func (api *PodPointAPI) saveAuthDetails() error {
	if api.refreshToken == "" || api.s.DataDir == "" {
		slog.Info("skipping", "r", api.refreshToken, "d", api.s.DataDir)
		return nil
	}

	payload := authPersistenceFile{
		Username:     api.s.PodPointUsername,
		RefreshToken: api.refreshToken,
	}

	payloadMarshalled, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling persisted auth details: %w", err)
	}

	err = os.WriteFile(path.Join(api.s.DataDir, "auth.json"), payloadMarshalled, 0o600)
	if err != nil {
		return fmt.Errorf("error writing persisted auth details: %w", err)
	}

	return nil
}

func (api *PodPointAPI) loadSavedAuthDetails() error {
	if api.s.DataDir == "" {
		slog.Info("no data directory - continuing without persisted auth details")
		return nil
	}

	authFileBytes, err := os.ReadFile(path.Join(api.s.DataDir, "auth.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Info("auth persistence file does not exist - continuing without it")
			return nil
		} else {
			return fmt.Errorf("error reading persisted auth details: %w", err)
		}
	}

	var authFile authPersistenceFile
	err = json.Unmarshal(authFileBytes, &authFile)
	if err != nil {
		return fmt.Errorf("error parsing persisted auth details: %w", err)
	}

	if authFile.Username != api.s.PodPointUsername {
		slog.Warn("persisted auth details were for a different username - they will be discarded")
		return nil
	}

	api.refreshToken = authFile.RefreshToken

	return nil
}

func (api *PodPointAPI) loadAuthToken() error {
	// already got a valid token?
	if api.apiToken != "" && time.Now().Before(api.apiTokenExpiry) {
		return nil
	}

	// got a refresh token?
	if api.refreshToken != "" {
		err := api.loadTokenViaRefresh()
		if err != nil {
			slog.Warn("error refreshing existing token - will try to get a new one", "error", err)
		} else {
			return nil
		}
	}

	// otherwise, fetch a token from scratch
	err := api.loadTokenViaLogin()
	if err != nil {
		return fmt.Errorf("error getting auth token: %w", err)
	}

	// save the refresh token for next time
	err = api.saveAuthDetails()
	if err != nil {
		slog.Warn("error saving auth details; this is not fatal but will mean every run creates a new login", "error", err)
	}

	return nil
}

func (api *PodPointAPI) loadTokenViaLogin() error {
	slog.Info("fetching new API token")

	payload := map[string]any{
		"email":             api.s.PodPointUsername,
		"password":          api.s.PodPointPassword,
		"returnSecureToken": true,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error building payload: %w", err)
	}

	req, err := api.unauthedReqeust("POST", googleLoginURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("error logging in: %w", err)
	}

	res, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("error logging in: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("error logging in: %s", res.Status)
	}

	type loginResponse struct {
		Kind         string `json:"kind"`
		APIToken     string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    string `json:"expiresIn"`
	}

	var resParsed loginResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&resParsed)
	if err != nil {
		return fmt.Errorf("error parsing login response: %w", err)
	}

	expiresInParsed, err := strconv.Atoi(resParsed.ExpiresIn)
	if err != nil {
		return fmt.Errorf("error parsing login response: expiry '%s' could not be converted to an int", resParsed.ExpiresIn)
	}

	if resParsed.Kind != "identitytoolkit#VerifyPasswordResponse" {
		return fmt.Errorf("unsupported login response '%s' (maybe MFA?)", resParsed.Kind)
	}

	api.apiToken = resParsed.APIToken
	api.apiTokenExpiry = time.Now().Add(time.Second * time.Duration(expiresInParsed)).Add(time.Second * -10)
	api.refreshToken = resParsed.RefreshToken

	return nil
}

func (api *PodPointAPI) loadTokenViaRefresh() error {
	slog.Info("refreshing API token")

	payload := fmt.Sprintf("grant_type=refresh_token&refresh_token=%s", api.refreshToken)

	req, err := api.unauthedReqeust("POST", googleRefreshURL, bytes.NewReader([]byte(payload)))
	if err != nil {
		return fmt.Errorf("error logging in: %w", err)
	}
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("error refreshing auth token: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("error refreshing auth token: %s", res.Status)
	}

	type refreshResponse struct {
		APIToken     string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    string `json:"expires_in"`
	}

	var resParsed refreshResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&resParsed)
	if err != nil {
		return fmt.Errorf("error parsing token refresh response: %w", err)
	}

	expiresInParsed, err := strconv.Atoi(resParsed.ExpiresIn)
	if err != nil {
		return fmt.Errorf("error parsing token refresh response: expiry '%s' could not be converted to an int", resParsed.ExpiresIn)
	}

	api.apiToken = resParsed.APIToken
	api.apiTokenExpiry = time.Now().Add(time.Second * time.Duration(expiresInParsed)).Add(time.Second * -10)
	api.refreshToken = resParsed.RefreshToken

	return nil
}
