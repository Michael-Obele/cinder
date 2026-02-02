# Queue + Worker Architecture (BullMQ)

## Goals
- Provide async crawl processing parity with Asynq.
- Maintain monolith deployment (API + worker in one container).
- Avoid stalled jobs under CPU-heavy scraping workloads.

## BullMQ Worker Model (Key Points)
- Workers pull jobs from Redis queues and process them asynchronously.
- Worker process functions are async and can report progress.
- Concurrency can be set per worker or by running multiple workers.

### Concurrency Options
1. **Single Worker, High Concurrency**
   - Uses event loop concurrency for IO-heavy tasks.
2. **Multiple Workers**
   - Runs several worker processes/threads for higher availability.
3. **Sandboxed Processors**
   - Isolate heavy work in separate processes.
   - Optional `useWorkerThreads` to run processors via worker threads.

## Monolith Pattern (API + Worker Thread)
- Start BullMQ worker in a **worker thread** to avoid blocking the API event loop.
- Shared Redis connection strategy:
  - Use independent Redis connections per worker (avoid shared connection contention).

## Graceful Shutdown Strategy
1. Stop accepting new HTTP requests.
2. Pause BullMQ worker (drain inflight jobs).
3. Close Playwright contexts and browser.
4. Close Redis connections.
5. Exit the worker thread.

## Retry + Backoff Plan
- Configure automatic retries for crawl jobs.
- Exponential backoff for transient failures (timeouts, 429s).
- Mark hard failures as dead-letter jobs with metadata for later review.

## Metrics to Track
- Queue depth
- Average job processing time
- Retry count and failure rates
- Memory per worker

## Source Notes
- BullMQ workers process jobs asynchronously and move them to `completed` or `failed` states.
- Concurrency is configured per worker or by scaling worker instances.
- Sandboxed processors can run in separate processes; worker threads can be enabled via `useWorkerThreads`.

**References**: BullMQ worker docs, concurrency docs, sandboxed processors docs.
