# Operations Runbook

> **Purpose:** Operational procedures for deploying and maintaining cinder-js  
> **Audience:** DevOps, on-call engineers  
> **Last Updated:** 2026-02-02

---

## Table of Contents

1. [Deployment Procedures](#deployment-procedures)
2. [Monitoring & Alerts](#monitoring--alerts)
3. [Common Issues & Troubleshooting](#common-issues--troubleshooting)
4. [Scaling Guidelines](#scaling-guidelines)
5. [Emergency Procedures](#emergency-procedures)
6. [Maintenance Tasks](#maintenance-tasks)

---

## Deployment Procedures

### First-Time Setup

#### 1. Leapcell Project Setup

```bash
# Login to Leapcell CLI
leapcell auth login

# Create new project
leapcell project create cinder-js

# Set secrets
leapcell secret set redis-url "rediss://user:pass@host:port"
leapcell secret set api-key "your-api-key"
```

#### 2. Deploy Application

```bash
# Deploy from repository
leapcell deploy --config leapcell.yaml

# Or deploy specific commit
leapcell deploy --commit abc123
```

#### 3. Verify Deployment

```bash
# Check deployment status
leapcell status cinder-js

# Test health endpoint
curl https://cinder-js.leapcell.app/health

# Test scrape endpoint
curl -X POST https://cinder-js.leapcell.app/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "mode": "static"}'
```

### Rolling Updates

```bash
# Deploy new version with zero downtime
leapcell deploy --rolling

# Monitor rollout
leapcell rollout status cinder-js

# Rollback if needed
leapcell rollback cinder-js --revision previous
```

### Blue-Green Deployment (Manual)

```bash
# Deploy new version to staging
leapcell deploy --environment staging

# Verify staging
curl https://cinder-js-staging.leapcell.app/health

# Promote to production
leapcell promote cinder-js --from staging --to production
```

---

## Monitoring & Alerts

### Key Metrics

| Metric       | Warning       | Critical      | Action                       |
| ------------ | ------------- | ------------- | ---------------------------- |
| Memory Usage | > 70% (2.8GB) | > 85% (3.4GB) | Scale up or investigate leak |
| CPU Usage    | > 70%         | > 90%         | Scale up                     |
| Error Rate   | > 1%          | > 5%          | Investigate logs             |
| P95 Latency  | > 4s          | > 8s          | Check browser pool           |
| Queue Depth  | > 50          | > 100         | Scale workers                |

### Health Endpoints

| Endpoint             | Purpose         | Expected Response                   |
| -------------------- | --------------- | ----------------------------------- |
| `GET /health`        | Liveness check  | `200 {"status": "healthy"}`         |
| `GET /health/ready`  | Readiness check | `200` or `503` if not ready         |
| `GET /health/detail` | Detailed status | Memory, queue stats, browser status |

### Log Analysis

```bash
# View recent logs
leapcell logs cinder-js --tail 100

# Filter by level
leapcell logs cinder-js --filter level=error

# Search for specific errors
leapcell logs cinder-js --grep "ECONNREFUSED"
```

### Sample Alert Rules (Prometheus)

```yaml
groups:
  - name: cinder-js-alerts
    rules:
      - alert: HighMemoryUsage
        expr: process_resident_memory_bytes > 3.4e9
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High memory usage on cinder-js"
          description: "Memory usage is {{ $value | humanize }} (> 3.4GB)"
          
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate on cinder-js"
          
      - alert: QueueBacklog
        expr: bullmq_queue_waiting > 100
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Queue backlog building up"
```

---

## Common Issues & Troubleshooting

### Issue: Out of Memory (OOM) Kills

**Symptoms:**
- Container restarts unexpectedly
- Logs show termination with exit code 137
- Memory metrics spike to 100%

**Diagnosis:**
```bash
# Check for OOM events
leapcell events cinder-js --type OOMKilled

# Check memory trend
leapcell metrics cinder-js --metric memory --period 1h
```

**Resolution:**
1. Check for concurrent dynamic scrape count
2. Verify browser contexts are being closed
3. Consider browser restart after N requests:
   ```typescript
   // Add BROWSER_RESTART_THRESHOLD env var
   if (requestCount > BROWSER_RESTART_THRESHOLD) {
     await browser.close();
     browser = await chromium.launch();
   }
   ```
4. Temporarily scale to higher memory tier

---

### Issue: Slow Cold Starts

**Symptoms:**
- First request times out
- Health checks fail during startup
- Container marked unhealthy

**Diagnosis:**
```bash
# Check startup time
leapcell logs cinder-js --since 5m | grep "startup"

# Check if browser initialization is bottleneck
leapcell logs cinder-js | grep "browser"
```

**Resolution:**
1. Enable lazy browser initialization (only init on first dynamic request)
2. Add keep-warm pings:
   ```bash
   # Cron job to ping every 5 minutes
   */5 * * * * curl -s https://cinder-js.leapcell.app/health > /dev/null
   ```
3. Increase min_instances in leapcell.yaml to 1+

---

### Issue: Browser Crashes

**Symptoms:**
- "Browser has been closed" errors
- Playwright errors in logs
- Dynamic scrapes failing

**Diagnosis:**
```bash
# Check for browser errors
leapcell logs cinder-js | grep -i "browser\|playwright\|chromium"

# Check memory at time of crash
leapcell metrics cinder-js --metric memory --since 30m
```

**Resolution:**
1. Implement browser reconnection:
   ```typescript
   async function getBrowser() {
     if (!browser || !browser.isConnected()) {
       browser = await chromium.launch();
     }
     return browser;
   }
   ```
2. Add browser health check and auto-restart
3. Reduce concurrent context limit

---

### Issue: Redis Connection Failures

**Symptoms:**
- Queue operations failing
- "ECONNREFUSED" or "ETIMEDOUT" in logs
- Caching not working

**Diagnosis:**
```bash
# Check Redis connectivity
leapcell logs cinder-js | grep -i "redis\|ioredis"

# Test Redis from container
leapcell exec cinder-js -- redis-cli -u $REDIS_URL ping
```

**Resolution:**
1. Verify REDIS_URL secret is set correctly
2. Check Redis provider (Upstash) status
3. Verify TLS configuration:
   ```typescript
   // Ensure TLS is enabled for rediss:// URLs
   const connection = new IORedis(process.env.REDIS_URL, {
     tls: process.env.REDIS_URL?.startsWith('rediss://') ? {} : undefined,
   });
   ```
4. Check for connection pool exhaustion

---

### Issue: Slow Queue Processing

**Symptoms:**
- Queue backlog growing
- Jobs timing out
- Completed jobs taking too long

**Diagnosis:**
```bash
# Check queue metrics
curl https://cinder-js.leapcell.app/admin/queue/stats

# Check worker status
leapcell logs cinder-js | grep "worker"
```

**Resolution:**
1. Increase worker concurrency (if memory allows):
   ```typescript
   const worker = new Worker('scrape', processor, {
     concurrency: 15, // up from default 10
   });
   ```
2. Check for job processing bottlenecks
3. Scale to additional instances

---

## Scaling Guidelines

### When to Scale Up

| Condition                    | Action                                  |
| ---------------------------- | --------------------------------------- |
| Memory > 70% sustained       | Add memory (4GB → 8GB) or add instance  |
| Queue depth > 50 for > 5 min | Add worker instance                     |
| P95 latency > 5s sustained   | Add instance or optimize                |
| Error rate > 2%              | Investigate first, then scale if needed |

### When to Scale Down

| Condition                | Action                            |
| ------------------------ | --------------------------------- |
| Memory < 30% for 1 hour  | Reduce memory allocation          |
| < 10 requests per minute | Reduce to min instances           |
| Queue empty for 1 hour   | Consider scale to 0 (with warmup) |

### Scaling Commands

```bash
# Scale instances manually
leapcell scale cinder-js --instances 3

# Update resource limits
leapcell config set cinder-js resources.memory=8GB

# Enable autoscaling
leapcell autoscale cinder-js \
  --min 1 \
  --max 5 \
  --cpu-target 70
```

---

## Emergency Procedures

### Complete Outage

1. **Check Leapcell status:** https://status.leapcell.io
2. **Check container status:**
   ```bash
   leapcell status cinder-js --verbose
   ```
3. **Force restart:**
   ```bash
   leapcell restart cinder-js --force
   ```
4. **If still failing, rollback:**
   ```bash
   leapcell rollback cinder-js --revision previous
   ```
5. **Notify stakeholders via incident channel**

### Redis Outage

1. **Verify Redis status with provider**
2. **Cinder can operate without Redis (no caching/queue):**
   - Static scrapes will still work
   - Dynamic scrapes will still work
   - Crawl (queue) endpoints will return 503
3. **If using Upstash, check their status page**
4. **Failover to backup Redis if configured**

### Security Incident

1. **Rotate secrets immediately:**
   ```bash
   leapcell secret set api-key "new-secure-key"
   leapcell restart cinder-js
   ```
2. **Check audit logs:**
   ```bash
   leapcell logs cinder-js --since 24h | grep -i "auth\|api-key"
   ```
3. **Disable external access if needed:**
   ```bash
   leapcell config set cinder-js networking.external=false
   ```

---

## Maintenance Tasks

### Weekly

- [ ] Review error logs for patterns
- [ ] Check memory trends
- [ ] Verify backup Redis connection works
- [ ] Update dependencies (security patches)

### Monthly

- [ ] Rotate API keys
- [ ] Review and optimize slow queries
- [ ] Update Playwright/Chromium versions
- [ ] Review resource allocation vs usage
- [ ] Update User-Agent lists

### Quarterly

- [ ] Performance benchmark comparison
- [ ] Cost analysis and optimization
- [ ] Security audit
- [ ] Disaster recovery test

### Dependency Update Procedure

```bash
# Create feature branch
git checkout -b update-deps

# Update dependencies
bun update

# Test locally
bun test
docker build -t cinder-js:test .
docker run -p 8080:8080 cinder-js:test

# Run integration tests
bun run test:integration

# Deploy to staging
leapcell deploy --environment staging

# Verify staging
# ... run smoke tests ...

# Merge and deploy to production
git checkout main
git merge update-deps
leapcell deploy
```

---

*Document Version: 1.0.0-draft*  
*Last Updated: 2026-02-02*
