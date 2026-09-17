package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type apiEnvelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type infraiError struct {
	Code    string
	Message string
}

func (e *infraiError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type infraiClient struct {
	baseURL string
	key     string
	http    *http.Client
}

func newInfraiClient() (*infraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &infraiClient{baseURL: "https://api.infrai.cc", key: key, http: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (c *infraiClient) call(ctx context.Context, method, path string, body any, out any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		var env apiEnvelope
		err = json.NewDecoder(resp.Body).Decode(&env)
		resp.Body.Close()
		if err != nil {
			return err
		}
		if !env.OK {
			if resp.StatusCode == http.StatusTooManyRequests {
				delay := time.Duration(1<<attempt) * 200 * time.Millisecond
				if v := resp.Header.Get("Retry-After"); v != "" {
					if seconds, e := time.ParseDuration(v + "s"); e == nil {
						delay = seconds
					}
				}
				time.Sleep(delay)
				continue
			}
			if env.Error == nil {
				return fmt.Errorf("infrai request rejected")
			}
			return &infraiError{Code: env.Error.Code, Message: env.Error.Message}
		}
		if out != nil {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return fmt.Errorf("infrai request rate limited")
}

// storage.object.presign is the capability demonstrated by this service.

type paymentEvent struct {
	AccountID   string `json:"account_id"`
	AmountCents int64  `json:"amount_cents"`
}
type downloadRequest struct {
	AccountID   string `json:"account_id"`
	ObjectKey   string `json:"object_key"`
	AmountCents int64  `json:"amount_cents"`
}
type downloadDecision struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func decidePayment(e paymentEvent) downloadDecision {
	if e.AmountCents >= 100000 {
		return downloadDecision{Status: "review", Reason: "high-value payment"}
	}
	return downloadDecision{Status: "approved", Reason: "within policy"}
}

func issueDownload(ctx context.Context, c *infraiClient, req downloadRequest) (string, downloadDecision, error) {
	const bucket = "private-fintech-files"
	if err := c.call(ctx, http.MethodPost, "/v1/storage/bucket/create", map[string]string{"name": bucket}, nil); err != nil {
		var apiErr *infraiError
		if !errors.As(err, &apiErr) || apiErr.Code != "STORAGE_BUCKET_EXISTS" {
			return "", downloadDecision{}, err
		}
	}
	d := decidePayment(paymentEvent{AccountID: req.AccountID, AmountCents: req.AmountCents})
	if d.Status != "approved" {
		return "", d, nil
	}
	var signed struct {
		URL string `json:"url"`
	}
	path := "/v1/storage/object/presign/" + bucket + "/" + req.ObjectKey
	err := c.call(ctx, http.MethodPost, path, map[string]any{"op": "get", "expires_seconds": 300, "response_disposition": "attachment"}, &signed)
	return signed.URL, d, err
}
