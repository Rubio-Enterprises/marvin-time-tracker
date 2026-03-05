package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"
)

const (
	apnsPushType = "liveactivity"
	attributesType = "TimeTrackerAttributes"
)

type APNsClient struct {
	client *apns2.Client
	topic  string
}

func NewAPNsClient(keyPath, keyID, teamID, bundleID string) (*APNsClient, error) {
	authKey, err := token.AuthKeyFromFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load APNs key: %w", err)
	}

	tok := &token.Token{
		AuthKey: authKey,
		KeyID:   keyID,
		TeamID:  teamID,
	}

	client := apns2.NewTokenClient(tok).Production()

	return &APNsClient{
		client: client,
		topic:  bundleID + ".push-type.liveactivity",
	}, nil
}

func (ac *APNsClient) StartActivity(pushToStartToken string, taskTitle string, startedAtMs int64) error {
	startedAt := time.UnixMilli(startedAtMs).UTC()

	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"timestamp":  time.Now().Unix(),
			"event":      "start",
			"content-state": map[string]interface{}{
				"taskTitle":  taskTitle,
				"startedAt":  startedAt.Format(time.RFC3339),
				"isTracking": true,
			},
			"attributes-type": attributesType,
			"attributes":      map[string]interface{}{},
			"alert": map[string]interface{}{
				"title": "Tracking Started",
				"body":  taskTitle,
			},
		},
	}

	return ac.send(pushToStartToken, payload, 10)
}

func (ac *APNsClient) UpdateActivity(updateToken string, taskTitle string, startedAtMs int64) error {
	startedAt := time.UnixMilli(startedAtMs).UTC()

	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"event":     "update",
			"content-state": map[string]interface{}{
				"taskTitle":  taskTitle,
				"startedAt":  startedAt.Format(time.RFC3339),
				"isTracking": true,
			},
		},
	}

	return ac.send(updateToken, payload, 10)
}

func (ac *APNsClient) EndActivity(updateToken string) error {
	dismissalDate := time.Now().Add(5 * time.Minute).Unix()

	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"timestamp":      time.Now().Unix(),
			"event":          "end",
			"dismissal-date": dismissalDate,
			"content-state": map[string]interface{}{
				"taskTitle":  "",
				"startedAt":  time.Now().UTC().Format(time.RFC3339),
				"isTracking": false,
			},
		},
	}

	return ac.send(updateToken, payload, 10)
}

func (ac *APNsClient) send(deviceToken string, payload map[string]interface{}, priority int) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	notification := &apns2.Notification{
		DeviceToken: deviceToken,
		Topic:       ac.topic,
		Payload:     payloadBytes,
		Priority:    priority,
		PushType:    apns2.EPushType(apnsPushType),
	}

	resp, err := ac.client.Push(notification)
	if err != nil {
		return fmt.Errorf("APNs push error: %w", err)
	}

	if !resp.Sent() {
		return fmt.Errorf("APNs rejected: %d %s", resp.StatusCode, resp.Reason)
	}

	log.Printf("APNs: sent %s to %s...%s", payload["aps"].(map[string]interface{})["event"], deviceToken[:8], deviceToken[len(deviceToken)-4:])
	return nil
}

// marshalAPNsPayload is exported for testing payload structure.
func marshalAPNsPayload(event string, taskTitle string, startedAtMs int64, isTracking bool) ([]byte, error) {
	startedAt := time.UnixMilli(startedAtMs).UTC()

	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"event":     event,
			"content-state": map[string]interface{}{
				"taskTitle":  taskTitle,
				"startedAt":  startedAt.Format(time.RFC3339),
				"isTracking": isTracking,
			},
		},
	}

	if event == "start" {
		aps := payload["aps"].(map[string]interface{})
		aps["attributes-type"] = attributesType
		aps["attributes"] = map[string]interface{}{}
		aps["alert"] = map[string]interface{}{
			"title": "Tracking Started",
			"body":  taskTitle,
		}
	}

	if event == "end" {
		aps := payload["aps"].(map[string]interface{})
		aps["dismissal-date"] = time.Now().Add(5 * time.Minute).Unix()
	}

	return json.Marshal(payload)
}
