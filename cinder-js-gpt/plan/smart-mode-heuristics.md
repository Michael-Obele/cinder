# Smart Mode Heuristics (Design Spec)

## Goal
Select the **lightest** scraping strategy that returns meaningful content while preserving correctness.

### Fallback Chain
```
fetch (fastest, no parsing) → Cheerio (static parse) → Playwright (full browser)
```

## Heuristic Signals
### Fetch-Level Signals
- **HTTP status** not 200–399 → immediate fallback.
- **Response size** < 4 KB → likely shell page.
- **Content-Type** not HTML → fail fast unless user requests raw HTML.

### Static HTML Signals (Cheerio)
- **Text density**: `textChars / htmlChars` < **0.02** suggests empty shell.
- **Meaningful text length**: `< 400 chars` after trimming → likely client-rendered.
- **Root container patterns** (any match increases dynamic score):
  - `id="root"`, `id="app"`, `id="__next"`
  - `data-reactroot`, `data-reactid`
  - `ng-version`, `ng-app`
  - `data-v-app`, `data-vue-meta`
- **Script-heavy HTML**: `<script>` tags > **25%** of DOM nodes.
- **SSR markers**: `window.__INITIAL_STATE__`, `__NEXT_DATA__`, `__NUXT__`.

### DOM Quality Signals
- **Main content selector missing** (no `<main>`, `<article>`, or header/body ratio < 0.1).
- **Link density** extremely high with low text (likely nav shell).

## Decision Logic (Scored)
- Start with **score = 0**
- Each dynamic signal adds **+1**
- If `score >= 3`, switch to Playwright
- If `score == 2`, retry static with extended wait or HTML snapshot rules

## Timeout Strategy
- **fetch timeout**: 5–7s
- **Cheerio parse timeout**: 1–2s (fail fast)
- **Playwright**: 15–30s hard cap

## Fallback Triggers
- HTML parse error or non-HTML response
- Missing `<body>` or empty `<body>`
- Page produces < 400 chars of visible text

## Output Consistency
- Always return the same response envelope with `engine` field
- Include `engine: fetch|cheerio|playwright`

## Validation Checklist
- Compare smart output against explicit `static` and `dynamic`
- Ensure `smart` never returns empty markdown when `dynamic` succeeds

