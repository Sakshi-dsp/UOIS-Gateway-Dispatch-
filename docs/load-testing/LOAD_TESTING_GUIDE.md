# Load Testing Guide

This guide explains how to perform load testing for the UOIS Gateway to validate throughput requirements.

## Requirements

- **Throughput**: Minimum 1000 requests/second
- **Latency SLOs**:
  - `/ondc/search`: < 500ms (p95)
  - `/ondc/confirm`: < 1s (p95)
  - `/ondc/status`: < 200ms (p95)
  - Callbacks: < 2s (p95)

## Prerequisites

1. Install [Vegeta](https://github.com/tsenart/vegeta):
   ```bash
   go install github.com/tsenart/vegeta/v12@latest
   ```

2. Ensure the gateway is running and accessible

3. Have valid client credentials configured

## Running Load Tests

### Using Bash Script (Linux/Mac)

```bash
export GATEWAY_URL=http://localhost:8080
export CLIENT_ID=your-client-id
export CLIENT_SECRET=your-client-secret
export ENDPOINT=/ondc/status
export RATE=1000
export DURATION=60s

./scripts/load_test.sh
```

### Using PowerShell Script (Windows)

```powershell
$env:GATEWAY_URL = "http://localhost:8080"
$env:CLIENT_ID = "your-client-id"
$env:CLIENT_SECRET = "your-client-secret"

.\scripts\load_test.ps1 -Endpoint "/ondc/status" -Rate 1000 -Duration "60s"
```

### Manual Vegeta Commands

1. Create a target file (`targets.txt`):
   ```
   POST http://localhost:8080/ondc/status
   Authorization: Basic <base64-encoded-credentials>
   Content-Type: application/json

   {
     "context": {
       "domain": "nic2004:60221",
       "country": "IND",
       "city": "std:080",
       "action": "status",
       "core_version": "1.2.0",
       "bap_id": "test-bap",
       "bap_uri": "https://test-bap.com",
       "transaction_id": "test-txn-123",
       "message_id": "test-msg-123",
       "timestamp": "2024-01-01T00:00:00.000Z",
       "ttl": "PT30S"
     },
     "message": {
       "order_id": "test-order-123"
     }
   }
   ```

2. Run the attack:
   ```bash
   vegeta attack -rate=1000 -duration=60s -targets=targets.txt | vegeta report
   ```

3. Generate latency histogram:
   ```bash
   vegeta attack -rate=1000 -duration=60s -targets=targets.txt | \
     vegeta report -type=hist[0,100ms,200ms,500ms,1s,2s]
   ```

## Interpreting Results

### Key Metrics

- **Requests**: Total number of requests sent
- **Rate**: Actual request rate achieved
- **Success Rate**: Percentage of successful responses (200-299)
- **Latency (p50, p95, p99)**: Percentile latencies
- **Throughput**: Requests per second

### Success Criteria

- ✅ Throughput >= 1000 req/s sustained
- ✅ p95 latency within SLO for each endpoint
- ✅ Success rate >= 99.9%
- ✅ No errors or timeouts

### Example Output

```
Requests      [total, rate]            60000, 1000.00
Duration      [total, attack, wait]     60.000s, 60.000s, 0.000s
Latencies     [min, mean, 50, 90, 95, 99, max]  10ms, 50ms, 45ms, 80ms, 120ms, 200ms, 500ms
Bytes In      [total, mean]             12000000, 200.00
Bytes Out     [total, mean]             18000000, 300.00
Success       [ratio]                   1.0000
Status Codes  [code:count]              200:60000
```

## Horizontal Scaling

The gateway is designed for horizontal scaling:

1. **Load Balancer**: Deploy multiple gateway instances behind a load balancer
2. **Stateless Design**: All instances share Redis and Postgres-E
3. **Scaling**: Add instances to increase throughput capacity

### Scaling Test

Test with multiple instances:

```bash
# Test with 2 instances (2000 req/s total)
vegeta attack -rate=2000 -duration=60s -targets=targets.txt | vegeta report
```

## Monitoring During Load Tests

Monitor these metrics during load tests:

1. **Prometheus Metrics** (`/metrics` endpoint):
   - `uois_request_duration_seconds` - Request latency histogram
   - `uois_requests_total` - Request count by endpoint/status
   - `uois_errors_total` - Error count

2. **System Metrics**:
   - CPU usage
   - Memory usage
   - Network I/O
   - Database connection pool

3. **Application Logs**:
   - Error rates
   - Timeout occurrences
   - Circuit breaker state

## Troubleshooting

### Low Throughput

- Check CPU/memory limits
- Verify database connection pool size
- Review Redis connection pool
- Check network bandwidth

### High Latency

- Review slow query logs
- Check Redis latency
- Verify gRPC service latency
- Review event processing lag

### Errors

- Check authentication/authorization
- Verify rate limiting configuration
- Review circuit breaker state
- Check dependency health

