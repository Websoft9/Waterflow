# Metrics Package

This package defines Prometheus metrics for monitoring Waterflow API performance.

## Metrics

### `waterflow_http_requests_total`
Counter metric tracking total HTTP requests.

**Labels:**
- `path`: Request path (e.g., `/health`, `/v1/workflows/validate`)
- `method`: HTTP method (e.g., `GET`, `POST`)
- `status`: HTTP status code (e.g., `200`, `404`)

### `waterflow_http_request_duration_seconds`
Histogram metric tracking HTTP request duration.

**Labels:**
- `path`: Request path
- `method`: HTTP method

**Buckets:** Default Prometheus buckets (0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10)

## Usage

Metrics are automatically registered on package initialization and collected by the `middleware.Metrics` middleware.
