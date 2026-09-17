package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	client, err := newInfraiClient()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req downloadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		url, decision, err := issueDownload(r.Context(), client, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		status := http.StatusOK
		if decision.Status != "approved" {
			status = http.StatusAccepted
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]any{"account_id": req.AccountID, "object_key": req.ObjectKey, "decision": decision, "signed_url": url})
	})
	log.Println("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
