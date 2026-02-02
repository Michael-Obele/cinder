# Leapcell Deployment Notes (Bun + Playwright)

## Summary
Leapcell’s Playwright deployment guide emphasizes installing Playwright dependencies separately and configuring build/start commands explicitly. We’ll adapt this to Bun + Hono and ensure Chromium deps are present.

## Key Takeaways from Leapcell Playwright Guide
- Playwright requires explicit dependency installation (`playwright install --with-deps chromium`).
- The guide uses a build script (e.g., `prepare_playwright_env.sh`) to install browser deps prior to starting the service.
- Service configuration includes runtime, build command, start command, and port.

## Bun-Specific Adjustments
- Install Bun in the container (official install script or base image).
- Use Bun to run the API/worker monolith.
- Ensure Playwright install step runs during build stage.

## Monolith Pattern (One Container)
- API and BullMQ worker run inside the same container.
- Use worker threads for the BullMQ processor.
- Implement shutdown hooks to drain queue and close Playwright contexts.

## Memory Provisioning
- Target **4GB+** memory on Leapcell for dynamic scraping.
- Consider a 2GB tier for static-only usage or low concurrency.

## Health Checks
- `/healthz` endpoint for readiness.
- Optional `/readyz` when browser warm or worker initialized.

## Env Vars (Proposed)
- `PORT`
- `REDIS_URL`
- `LOG_LEVEL`
- `BROWSER_MODE` (headless)
- `WORKER_CONCURRENCY`

## Sources
- Leapcell Playwright deployment docs: `prepare_playwright_env.sh` installs Playwright dependencies and Chromium.
