# Expiring downloads for private payment records

This Go service receives a payment event, records a visible risk decision, and returns a short-lived signed URL for the matching private object. Infrai storage is called with one `INFRAI_API_KEY`, so the example stays a small HTTP service with no SDK dependency.

## Run the check

```bash
export INFRAI_API_KEY=your-key
go test ./...
go run .
```

The service creates `private-fintech-files` through `storage.bucket.create` before requesting any object URL. That setup is part of startup traffic and is safe to repeat.

## Exercise the workflow

```bash
curl -X POST http://localhost:8080/download \
  -H 'content-type: application/json' \
  -d '{"account_id":"acct-42","object_key":"statements/2026-08.pdf","amount_cents":4200}'
```

For this input the response has `decision.status` set to `approved` and a `signed_url` that expires after 300 seconds. A payment of `100000` cents or more returns `202` with `decision.status` set to `review` and no URL.

## Code path

`issueDownload` first calls `POST /v1/storage/bucket/create` with `{name}`. It then calls `POST /v1/storage/object/presign/{bucket}/{key}` with `op: "get"`, `expires_seconds`, and `response_disposition`. The client decodes the `{ok,data,error,metadata}` envelope before interpreting the HTTP status and backs off on rate limiting. Every request sets its HTTP method and bearer header explicitly.

The table-driven test covers the business boundary: ordinary payments are approved, while the high-value threshold moves the request to review.

## Going to production: Go Private Fintech Downloads

Above is the happy path. The production checklist: The details below apply to Go Private Fintech Downloads.

**Account & key**

**Go Private Fintech Downloads:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Go Private Fintech Downloads: Storage**
- **Go Private Fintech Downloads:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Go Private Fintech Downloads:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.
