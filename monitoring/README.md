# Monitoring Configuration

This directory contains operational monitoring configuration files for the UOIS Gateway.

## Files

- **`prometheus_alerts.yml`** - Prometheus alerting rules for SLO monitoring
  - Latency SLO alerts (per endpoint)
  - Availability SLO alerts (99.9% uptime)
  - Error rate alerts
  - Service health alerts
  - Circuit breaker state alerts

## Deployment

These files are meant to be deployed to your monitoring infrastructure:

1. **Prometheus Alerting Rules:**
   - Copy `prometheus_alerts.yml` to your Prometheus server's rules directory
   - Configure Prometheus to load rules from this file
   - Ensure Prometheus can scrape metrics from the gateway's `/metrics` endpoint

2. **Configuration:**
   - Update alerting channel configuration (email, Slack, PagerDuty) in your Prometheus/Alertmanager setup
   - Adjust evaluation intervals if needed (currently 5m for SLO alerts, 1m for health alerts)

## Documentation

For monitoring setup and usage documentation, see:
- `docs/load-testing/LOAD_TESTING_GUIDE.md` - Load testing and performance validation
- `docs/production-docs/UOISGateway_FR.md` - Section 12 (Non-Functional Requirements)

