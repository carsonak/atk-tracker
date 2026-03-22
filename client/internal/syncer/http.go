package syncer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"atk-tracker/shared/go/atkshared"
)

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func New(baseURL string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
	}
}

func (h *HTTPClient) CreateSession(ctx context.Context, apprenticeID, machineID string) (string, error) {
	body := atkshared.CreateSessionRequest{ApprenticeID: apprenticeID, MachineID: machineID}
	resp, err := h.sendJSON(ctx, http.MethodPost, h.baseURL+"/sessions", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create session failed status=%d", resp.StatusCode)
	}
	var out atkshared.CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.SessionID, nil
}

func (h *HTTPClient) EndSession(ctx context.Context, sessionID string, end time.Time) error {
	resp, err := h.sendJSON(ctx, http.MethodPut, h.baseURL+"/sessions/"+sessionID+"/end", atkshared.EndSessionRequest{EndTime: end.UTC()})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("end session failed status=%d", resp.StatusCode)
	}
	return nil
}

func (h *HTTPClient) SendHeartbeat(ctx context.Context, payload atkshared.HeartbeatPayload) error {
	resp, err := h.sendJSON(ctx, http.MethodPost, h.baseURL+"/heartbeats", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		return nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound {
		return ErrSessionInvalid
	}
	return fmt.Errorf("heartbeat failed status=%d", resp.StatusCode)
}

var ErrSessionInvalid = fmt.Errorf("session invalid")

func (h *HTTPClient) sendJSON(ctx context.Context, method, url string, body interface{}) (*http.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return h.client.Do(req)
}
