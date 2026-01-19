# Setup & Usage

## 1. Start Cinder API

Ensure the Cinder Go backend is running.

```bash
# From the root of the cinder repo
go run cmd/api/main.go
```

_Note: Ensure Redis is running if you plan to use Crawl features._

## 2. Build Mastra MCP

```bash
cd mastra
npm install
npm run build
```

## 3. Connect to MCP Client

You can use this MCP server with any MCP-compatible client (Claude Desktop, Cursor, etc.).

### Claude Desktop Configuration

Add the following to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "cinder": {
      "command": "node",
      "args": ["/absolute/path/to/cinder/mastra/dist/mastra/index.js"],
      "env": {
        "CINDER_API_URL": "http://localhost:8080"
      }
    }
  }
}
```

### Testing

Once connected, you can ask the AI:

- "Scrape https://example.com and tell me the main heading."
- "Search for 'Golang web scraping libraries' using Cinder."
- "Start a crawl for https://news.ycombinator.com."
