package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDecidePayment(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		want   string
	}{{"ordinary", 4200, "approved"}, {"review threshold", 100000, "review"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decidePayment(paymentEvent{AmountCents: tt.amount})
			if got.Status != tt.want {
				t.Fatalf("status=%q want %q", got.Status, tt.want)
			}
		})
	}
}

func TestIssueDownloadContinuesWhenBucketExists(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/storage/bucket/create":
			w.WriteHeader(http.StatusConflict)
			fmt.Fprint(w, `{"ok":false,"error":{"code":"STORAGE_BUCKET_EXISTS","message":"bucket exists"}}`)
		case "/v1/storage/object/presign/private-fintech-files/statement.pdf":
			fmt.Fprint(w, `{"ok":true,"data":{"url":"https://download.example/statement.pdf"}}`)
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client := &infraiClient{baseURL: server.URL, key: "test", http: server.Client()}
	url, decision, err := issueDownload(context.Background(), client, downloadRequest{ObjectKey: "statement.pdf", AmountCents: 4200})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || decision.Status != "approved" || url != "https://download.example/statement.pdf" {
		t.Fatalf("requests=%d decision=%q url=%q", requests, decision.Status, url)
	}
}
