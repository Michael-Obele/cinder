# 🛠️ Svelte Developer Workflow: Run, Use, Test & Debug

Welcome to the day-to-day workflow guide! As a Svelte developer working on Cinder, you're bridging the gap between a robust Go backend and a sleek Svelte/JS frontend ecosystem. This guide tells you exactly how to spin things up, handle integrations, and troubleshoot when things go wrong.

---

## 🏃 1. How to Run the Project

The Cinder project uses a **monolith pattern** for local development. You do not need to start up five different microservices.

### Running the Go Backend

The backend contains both the API server and the background worker.

```bash
# From the project root
go run cmd/api/main.go
```

**What this does:**
- Starts the HTTP API on `http://localhost:8080`
- Automatically starts the background worker (listening to Redis) within the same process.
- Hot-reloading is not built into Go by default (unlike Vite). If you make a change to a `.go` file, you need to stop (`Ctrl+C`) and re-run the command.
  - *Pro-tip:* Install `air` (`go install github.com/cosmtrek/air@latest`) and just run `air` in the terminal for hot-reloading!

### Running the Frontend / JS Services

If you are working in `cinder-js` or a SvelteKit consuming app:

```bash
# In your Svelte/JS directory
npm install
npm run dev
```

This runs your standard Vite dev server, typically on `http://localhost:5173`. 

---

## 💻 2. How to Use (Consuming the API)

As a Svelte developer, your main interaction with the Go backend will be via `fetch` calls, typically inside `+page.server.ts` or `+server.ts` files.

### Example: Calling the Scrape API from SvelteKit

```typescript
// src/routes/dashboard/+page.server.ts
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
    // 1. Hit the local Go backend
    const response = await fetch('http://localhost:8080/v1/scrape', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            url: 'https://example.com',
            mode: 'smart' // 'static', 'dynamic', or 'smart'
        })
    });

    if (!response.ok) {
        // Handle Go error responses
        const errorData = await response.json();
        console.error("Cinder API Error:", errorData);
        return { error: 'Failed to scrape' };
    }

    const data = await response.json();
    
    // 2. Pass the Markdown/HTML to the Svelte component
    return {
        markdown: data.markdown,
        metadata: data.metadata
    };
};
```

---

## 🧪 3. How to Test

You live in two worlds: the Go backend and the JS/Svelte frontend.

### Testing the Go Backend

There is no Jest or Vitest here. Go has built-in testing.

```bash
# Run all backend tests across the project
go test ./... -v

# Run tests without caching (if you suspect stale results)
go test ./... -count=1 -v

# Run a specific test suite (e.g., scraper package)
go test ./internal/scraper/... -v
```

**Mental mapping:**
- `describe()` / `it()` -> `func TestSomething(t *testing.T) { t.Run(...) }`
- `expect(x).toBe(y)` -> `if x != y { t.Errorf(...) }`

*For more in-depth testing setup, check out the [Testing Guide](TESTING.md).*

### Testing the Svelte/JS Side

Business as usual!
```bash
# In your Svelte project
npm run test:unit      # Vitest
npm run test:ui        # Vitest UI
npm run test:e2e       # Playwright
```

---

## 🐛 4. How to Debug

Debugging cross-stack can be tricky. Here is how to find out why things are breaking.

### Debugging Go with VS Code

Don't just use `fmt.Println` (the Go equivalent of `console.log`). Use the debugger!

1. Install the **Go extension** in VS Code.
2. Open the "Run and Debug" panel (Ctrl+Shift+D).
3. Click "create a launch.json file" and select "Go".
4. Replace the contents of `.vscode/launch.json` with:
   ```json
   {
       "version": "0.2.0",
       "configurations": [
           {
               "name": "Launch Cinder API",
               "type": "go",
               "request": "launch",
               "mode": "auto",
               "program": "${workspaceFolder}/cmd/api/main.go",
               "env": {
                   "LOG_LEVEL": "debug"
               }
           }
       ]
   }
   ```
5. Set breakpoints in your `.go` files by clicking the gutter.
6. Hit **F5**. The API server will start, and VS Code will pause execution on your breakpoints, allowing you to inspect variables just like in Chrome DevTools!

### Logging Fallback (`console.log` equivalent)

If you must "console.log" something quickly in Go, use the structured logger instead of `fmt.Println`:

```go
import "github.com/standard-user/cinder/pkg/logger"

// Equivalent to console.log("Data:", myVar)
logger.Log.Info("Debugging", "myVar", myVar)

// Equivalent to console.error("Error:", err)
logger.Log.Error("Something broke", "error", err)
```

Make sure your server is running with `LOG_LEVEL=debug` if you are using `logger.Log.Debug()`.

### Common Gotchas

- **CORS Errors**: If your SvelteKit frontend (running in the browser) tries to call `localhost:8080/v1/scrape` directly via a client-side `fetch()`, you might get CORS errors. **Always proxy requests through your SvelteKit `+server.ts` endpoints or use `+page.server.ts`** to make server-to-server calls.
- **Port Conflicts**: If `localhost:8080` is taken, set `PORT=8081` in your `.env` file for the Go backend.
- **Nil Pointers**: The equivalent of "Cannot read properties of undefined". If Go panics with "invalid memory address or nil pointer dereference", look for a variable that is a pointer (has a `*` in its type) that was never initialized.
