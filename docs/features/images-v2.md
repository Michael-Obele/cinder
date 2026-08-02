# Image Engine v2

> Feature spec — implemented 2026-08-01 (`internal/image`, `internal/scraper/service.go`)

## What changed

### 1. Modern source extraction (`internal/image/extractor.go`)

The extractor now understands how real sites ship images:

| Source                       | Support                                                                                                       |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `og:image` / `twitter:image` | Highest priority, unchanged                                                                                   |
| `img[src]`                   | Content images                                                                                                |
| `srcset` / `sizes`           | Picks the **largest** width descriptor (`480w 800w` → `800w`), falls back to highest density (`1x 2x` → `2x`) |
| `<picture><source srcset>`   | First source wins, per HTML spec                                                                              |
| Lazy loading                 | `data-srcset`, `data-src`, `data-lazy-src`, `data-original` used when `src` is a placeholder or `data:` URI   |
| Next.js optimizer            | `/_next/image?url=...&w=640` unwraps to the **origin image URL**; the `w` param feeds dimension-based ranking |
| CSS backgrounds              | `style="background-image: url(...)"` (capped at 5 per page)                                                   |
| Tracking pixels              | `pixel/tracking/beacon/analytics/1x1/spacer/blank` URLs skipped                                               |

### 2. Quality-ranked selection

`max_images` now returns the **best** images, not the first N in DOM order:

```
og:image       100
twitter:image   90
hero (≥300px)   75
content         40
background      30
avatar/icon/logo/≤128px   10
```

This fixes the real-world failure where `firecrawl.dev/blog` returned four 48px
author avatars instead of the hero image.

### 3. Dimension sniffing (`internal/image/dimensions.go`)

`SniffDimensions` reads image headers via `image.DecodeConfig` (PNG/JPEG/WebP)
with a 512KB cap — no full decode.

### 4. Optional processing (`internal/image/optimize.go`)

`image_process` resizes (aspect-preserving, never upscales) and re-encodes
blobs as JPEG (default) or PNG. WebP output is intentionally unsupported:
there is no pure-Go WebP encoder and cgo bindings conflict with hobby-tier
container builds.

## API

```json
{
  "url": "https://example.com",
  "images": true,
  "image_format": "blob",
  "max_images": 5,
  "max_image_size_kb": 5120,
  "image_process": { "format": "jpeg", "max_width": 800, "quality": 80 }
}
```

## Behavior notes

- Blob fetching is **parallel** (limit 5) with per-image failure isolation.
- `max_image_size_kb` is honored per image (was previously dead config).
- Results remain ordered by ranking, not fetch completion.
