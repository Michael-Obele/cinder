# Interactive API Documentation (Swagger)

Cinder provides interactive API documentation out-of-the-box using Swagger (via Swaggo). This allows you to view the API schema and test endpoints directly from your browser.

## Accessing the Swagger UI

1. Start the Cinder API server:
   ```bash
   go run cmd/api/main.go
   ```
2. Open your browser and navigate to the Swagger UI endpoint:
   ```http
   http://localhost:8080/swagger/index.html
   ```
   *(Adjust the port if you have configured Cinder to run on a port other than `8080`)*.

The Swagger interface allows you to view all available endpoints, required parameters, and response types. You can even execute actual test requests (e.g., triggering a `/v1/scrape`) directly against your local running instance.

---

## Updating the Swagger Documentation

The Swagger documentation is generated statically from annotations in the Go source code. If you add a new endpoint or change an existing request/response structure, you must regenerate the Swagger spec files.

### 1. Install Swag CLI
If you haven't already, install the `swag` command-line tool:
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

### 2. Add or Update Annotations
Cinder uses declarative `// @...` comments above handler functions.
For example, in `internal/api/handlers/scrape.go`:
```go
// @Summary      Scrape a webpage
// @Description  Scrapes a given URL and returns its markdown content.
// @Tags         scrape
// @Accept       json
// @Produce      json
// @Param        url    query     string  false  "The URL to scrape"
// @Success      200    {object}  domain.ScrapeResult
// @Router       /scrape [post]
func (h *ScrapeHandler) Scrape(c *gin.Context) {
    // ...
}
```

### 3. Generate the Files
Because Cinder's `main.go` and handlers are located in different directories, you must run the following exact command from the root of the repository to regenerate the docs:

```bash
~/go/bin/swag init -d ./cmd/api,./internal/api/handlers,./internal/domain -g main.go -o internal/api/docs --parseDependency --parseInternal
```

#### What this command does:
- `-d ./cmd/api,./internal/api/handlers,./internal/domain`: Instructs Swag to search these directories for annotations.
- `-g main.go`: Points to the main file containing the general API info (e.g., `@title`, `@version`).
- `-o internal/api/docs`: Outputs the resulting `swagger.json`, `swagger.yaml`, and `docs.go` files to this specific internal directory to keep the root clean.
- `--parseDependency --parseInternal`: Ensures that structures located outside standard scopes (like the domain models) are successfully resolved by the parser.

After running this command, simply restart your API server, and the new changes will be visible in the browser UI.
