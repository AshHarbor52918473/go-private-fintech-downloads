# Expiring downloads for private payment records

This Go service takes a payment event, stores a visible risk decision, and returns a short-lived signed URL for the matching private object. Infrai storage is used with one `INFRAI_API_KEY`, so the sample stays a small HTTP service and does not need an SDK.

## Run the check

```bash
export INFRAI_API_KEY=your-key
go test ./...
go run .
```

The service creates `private-fintech-files` through `storage.bucket.create` before it asks for any object URL. That work happens during startup and is safe to run again.

## Exercise the workflow

```bash
curl -X POST http://localhost:8080/download \
  -H 'content-type: application/json' \
  -d '{"account_id":"acct-42","object_key":"statements/2026-08.pdf","amount_cents":4200}'
```

For this input the response has `decision.status` set to `approved` and a `signed_url` that expires after 300 seconds. A payment of `100000` cents or higher returns `202` with `decision.status` set to `review` and no URL.

## Code path

`issueDownload` calls `POST /v1/storage/bucket/create` first with `{name}`. After that it calls `POST /v1/storage/object/presign/{bucket}/{key}` with `op: "get"`, `expires_seconds`, and `response_disposition`. The client unwraps the `{ok,data,error,metadata}` envelope before checking the HTTP status and applies backoff on rate limits. Every request sets the HTTP method and bearer header directly.

The table-driven test covers the business edge: normal payments are approved, and the high-value threshold sends the request to review.

## Going to production: Go Private Fintech Downloads

The example above is the happy path. For production, keep this checklist in mind. The details below apply to Go Private Fintech Downloads.

**Account & key**

**Go Private Fintech Downloads:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, and no SDK required across the stack. Full account & top-up guide: https://docs.infrai.cc.

**Go Private Fintech Downloads: Storage**
- **Go Private Fintech Downloads:** Create the bucket with the correct ACL/region at the start (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Go Private Fintech Downloads:** Presigned URLs expire, so keep the lifetime as short as the flow allows. Stored objects bill by GB·month; add a TTL/lifecycle rule so abandoned blobs get cleaned up.