# Anti-Detection Strategy

> **Purpose:** Document stealth scraping patterns and bot evasion techniques  
> **Goal:** Match capabilities of Go's undetected-chromedp  
> **Last Updated:** 2026-02-03

---

## Table of Contents

1. [Overview](#overview)
2. [Playwright-Extra Integration](#playwright-extra-integration)
3. [Stealth Plugin Configuration](#stealth-plugin-configuration)
4. [User-Agent Rotation](#user-agent-rotation)
5. [Additional Evasion Techniques](#additional-evasion-techniques)
6. [Detection Testing](#detection-testing)
7. [Known Limitations](#known-limitations)

---

## Overview

### The Detection Problem

Modern websites use various techniques to detect automated browsers:

| Detection Method       | What It Checks                   | Evasion Strategy          |
| ---------------------- | -------------------------------- | ------------------------- |
| **WebDriver flag**     | `navigator.webdriver === true`   | Override to `undefined`   |
| **Headless UA**        | "HeadlessChrome" in User-Agent   | Use standard Chrome UA    |
| **Chrome plugins**     | Empty `navigator.plugins` array  | Inject fake plugins       |
| **WebGL fingerprint**  | GPU vendor/renderer strings      | Spoof consistent values   |
| **Automation timings** | Inhuman click/type speeds        | Add human-like delays     |
| **Behavior analysis**  | Mouse movements, scroll patterns | Simulate natural behavior |

### Goal

Match the evasion capabilities of Go's `undetected-chromedp` library using `playwright-extra` with the stealth plugin.

---

## Playwright-Extra Integration

### Installation

```bash
bun add playwright-extra puppeteer-extra-plugin-stealth
```

**Note:** The stealth plugin is from `puppeteer-extra` but works with `playwright-extra` due to shared plugin architecture.

### Basic Setup

```typescript
// Conceptual implementation - not production code
import { chromium } from 'playwright-extra';
import StealthPlugin from 'puppeteer-extra-plugin-stealth';

// Use stealth plugin
chromium.use(StealthPlugin());

// Launch with stealth enabled
const browser = await chromium.launch({
  headless: true,
  args: [
    '--disable-blink-features=AutomationControlled',
    '--disable-extensions',
    '--no-sandbox',
    '--disable-setuid-sandbox',
  ],
});
```

### Why Playwright-Extra?

| Library              | Stealth Support | Maintenance | API         |
| -------------------- | --------------- | ----------- | ----------- |
| playwright (vanilla) | ❌ None          | ✅ Microsoft | ✅ Clean     |
| playwright-extra     | ✅ Plugin-based  | ✅ Active    | ✅ Clean     |
| puppeteer-extra      | ✅ Plugin-based  | ✅ Active    | ⚠️ Different |

`playwright-extra` wraps vanilla Playwright and adds plugin support, including the mature stealth plugin ecosystem.

---

## Stealth Plugin Configuration

### Default Evasions (Enabled Automatically)

The stealth plugin applies these evasions by default:

| Evasion                           | Description                     | Bot Detection Bypassed  |
| --------------------------------- | ------------------------------- | ----------------------- |
| **chrome.app**                    | Adds missing `chrome.app` API   | Basic checks            |
| **chrome.csi**                    | Adds `chrome.csi` function      | Chrome fingerprinting   |
| **chrome.loadTimes**              | Adds `chrome.loadTimes`         | Timing analysis         |
| **chrome.runtime**                | Fixes `chrome.runtime` behavior | Extension checks        |
| **iframe.contentWindow**          | Fixes iframe access             | Frame detection         |
| **media.codecs**                  | Reports realistic codec support | Media fingerprinting    |
| **navigator.hardwareConcurrency** | Overrides CPU count             | Hardware fingerprinting |
| **navigator.languages**           | Sets realistic languages        | Locale detection        |
| **navigator.permissions**         | Fixes permission queries        | Permission API checks   |
| **navigator.plugins**             | Injects fake plugin list        | Plugin detection        |
| **navigator.vendor**              | Sets correct vendor             | Basic UA checks         |
| **navigator.webdriver**           | Sets to `undefined`             | **Critical**            |
| **sourceurl**                     | Hides injected script URLs      | DevTools detection      |
| **user-agent-override**           | Patches UA consistently         | UA mismatch             |
| **webgl.vendor**                  | Spoofs WebGL strings            | GPU fingerprinting      |
| **window.outerdimensions**        | Realistic window dimensions     | Layout analysis         |

### Custom Configuration

```typescript
// Conceptual configuration
import StealthPlugin from 'puppeteer-extra-plugin-stealth';

// Create stealth plugin with custom options
const stealth = StealthPlugin();

// Disable specific evasions if causing issues
stealth.enabledEvasions.delete('chrome.runtime'); // Example

// Add to playwright-extra
chromium.use(stealth);
```

### Advanced: WebGL Fingerprint Spoofing

```typescript
// Spoof WebGL vendor and renderer to match common GPUs
const context = await browser.newContext({
  // These will be handled by stealth plugin, but can be customized
});

// Manual override if needed (via page.addInitScript)
await page.addInitScript(() => {
  const getParameter = WebGLRenderingContext.prototype.getParameter;
  WebGLRenderingContext.prototype.getParameter = function(param) {
    // UNMASKED_VENDOR_WEBGL
    if (param === 37445) {
      return 'Intel Inc.';
    }
    // UNMASKED_RENDERER_WEBGL
    if (param === 37446) {
      return 'Intel(R) Iris(TM) Plus Graphics 640';
    }
    return getParameter.call(this, param);
  };
});
```

---

## User-Agent Rotation

### Strategy

Rotate User-Agent strings to:
1. Avoid fingerprinting based on static UA
2. Match current browser versions
3. Maintain consistency with other headers

### Implementation

```typescript
// Common real Chrome User-Agents (update periodically)
const USER_AGENTS = [
  // Windows Chrome
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36',
  
  // macOS Chrome  
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36',
  
  // Linux Chrome
  'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
];

function getRandomUA(): string {
  return USER_AGENTS[Math.floor(Math.random() * USER_AGENTS.length)];
}
```

### Consistency Requirements

When using a User-Agent, ensure these headers match:

| Header               | Must Match             |
| -------------------- | ---------------------- |
| `User-Agent`         | Selected UA            |
| `sec-ch-ua`          | Chrome version from UA |
| `sec-ch-ua-platform` | OS from UA             |
| `sec-ch-ua-mobile`   | "?0" for desktop       |

```typescript
// Playwright handles this automatically with userAgent option
const context = await browser.newContext({
  userAgent: getRandomUA(),
  // Client hints are automatically derived
});
```

---

## Additional Evasion Techniques

### 1. Timezone Consistency

Match timezone with apparent location:

```typescript
const context = await browser.newContext({
  timezoneId: 'America/New_York',
  locale: 'en-US',
});
```

### 2. Viewport Variation

Use realistic, varied viewports:

```typescript
const VIEWPORTS = [
  { width: 1920, height: 1080 },
  { width: 1536, height: 864 },
  { width: 1366, height: 768 },
  { width: 1440, height: 900 },
  { width: 1280, height: 720 },
];

const viewport = VIEWPORTS[Math.floor(Math.random() * VIEWPORTS.length)];
const context = await browser.newContext({
  viewport,
});
```

### 3. Human-Like Delays

Add realistic delays for interactions:

```typescript
// Between page loads
await page.waitForTimeout(1000 + Math.random() * 2000);

// Before clicking
await page.waitForTimeout(100 + Math.random() * 300);

// Typing with realistic speed
await page.type('input', 'search query', { delay: 50 + Math.random() * 100 });
```

### 4. Mouse Movement Simulation

For sites with advanced detection:

```typescript
// Move mouse naturally before clicking
async function humanClick(page, selector) {
  const element = await page.$(selector);
  const box = await element.boundingBox();
  
  // Random offset within element
  const x = box.x + box.width * (0.3 + Math.random() * 0.4);
  const y = box.y + box.height * (0.3 + Math.random() * 0.4);
  
  // Move to position with slight arc
  await page.mouse.move(x, y, { steps: 10 + Math.floor(Math.random() * 10) });
  
  // Small delay before click
  await page.waitForTimeout(50 + Math.random() * 150);
  
  await page.mouse.click(x, y);
}
```

### 5. Launch Arguments

Optimal Chromium flags for stealth:

```typescript
const browser = await chromium.launch({
  headless: true,
  args: [
    // Disable automation detection
    '--disable-blink-features=AutomationControlled',
    
    // Disable extensions (common bot indicator)
    '--disable-extensions',
    
    // Disable dev shm usage (stability)
    '--disable-dev-shm-usage',
    
    // Disable GPU (consistency in headless)
    '--disable-gpu',
    
    // Security flags for Docker
    '--no-sandbox',
    '--disable-setuid-sandbox',
    
    // Disable popup blocking
    '--disable-popup-blocking',
    
    // Disable infobars
    '--disable-infobars',
    
    // Window size
    '--window-size=1920,1080',
  ],
});
```

---

## Detection Testing

### Test Sites

Use these sites to validate evasion effectiveness:

| Site                                                     | What It Tests               | Target Result          |
| -------------------------------------------------------- | --------------------------- | ---------------------- |
| [bot.sannysoft.com](https://bot.sannysoft.com)           | Comprehensive bot detection | All green ✅            |
| [fingerprintjs.com/demo](https://fingerprintjs.com/demo) | Browser fingerprinting      | Consistent fingerprint |
| [browserleaks.com](https://browserleaks.com)             | Various fingerprints        | Realistic values       |
| [amiunique.org](https://amiunique.org)                   | Uniqueness score            | Normal browser range   |
| [pixelscan.net](https://pixelscan.net)                   | Detailed analysis           | "Not a bot" result     |

### Automated Testing

```typescript
// Conceptual test suite
async function testStealthCapabilities() {
  const browser = await chromium.launch();
  const page = await browser.newPage();
  
  // Test bot.sannysoft.com
  await page.goto('https://bot.sannysoft.com');
  await page.waitForTimeout(5000);
  
  // Screenshot for analysis
  await page.screenshot({ path: 'stealth-test-result.png', fullPage: true });
  
  // Extract results
  const results = await page.evaluate(() => {
    const rows = document.querySelectorAll('table tr');
    const checks = {};
    rows.forEach(row => {
      const cells = row.querySelectorAll('td');
      if (cells.length >= 2) {
        const name = cells[0].textContent?.trim();
        const result = cells[1].textContent?.trim();
        if (name && result) {
          checks[name] = result;
        }
      }
    });
    return checks;
  });
  
  console.log('Stealth Test Results:', results);
  await browser.close();
}
```

### Expected Results

For a properly configured stealth setup:

| Check              | Expected                        |
| ------------------ | ------------------------------- |
| webdriver          | `missing (passed ✅)` or `false` |
| webdriver advanced | `passed ✅`                      |
| Chrome             | `present (passed ✅)`            |
| Permissions        | `passed ✅`                      |
| Plugins            | `present (passed ✅)`            |
| Languages          | `passed ✅`                      |
| WebGL Vendor       | Realistic GPU vendor            |
| WebGL Renderer     | Realistic GPU renderer          |

---

## Known Limitations

### What We Cannot Evade

| Detection Method         | Why It's Hard           | Current Status             |
| ------------------------ | ----------------------- | -------------------------- |
| **Cloudflare Turnstile** | CAPTCHA challenge       | ❌ Requires solving service |
| **reCAPTCHA v3**         | Behavioral analysis     | ⚠️ May fail on low scores   |
| **Datadome**             | Advanced fingerprinting | ⚠️ Partial success          |
| **PerimeterX**           | ML-based detection      | ⚠️ Partial success          |
| **Imperva**              | Enterprise-grade WAF    | ⚠️ Partial success          |
| **Custom ML solutions**  | Site-specific           | ❓ Varies                   |

### Mitigation for Hard Cases

1. **Proxy Rotation:** Use residential proxies to avoid IP reputation issues
2. **Captcha Services:** Integrate with 2captcha/anticaptcha for challenges
3. **Session Persistence:** Maintain cookies/storage between requests
4. **Rate Limiting:** Respect site limits to avoid triggering blocks

### Legal Considerations

⚠️ **Important:** This documentation is for legitimate scraping of public data. Always:
- Respect `robots.txt`
- Follow site Terms of Service
- Implement rate limiting
- Handle opt-out requests
- Consult legal advice for sensitive use cases

---

## Comparison with Go Implementation

### Current Go (undetected-chromedp) Capabilities

```go
// From Cinder Go implementation
chromedp.Flag("disable-blink-features", "AutomationControlled"),
chromedp.Flag("exclude-switches", "enable-automation"),
chromedp.UserAgent(randomUA),
```

### Equivalent JS Implementation

```typescript
// playwright-extra with stealth does MORE than Go version
chromium.use(StealthPlugin());

const browser = await chromium.launch({
  args: ['--disable-blink-features=AutomationControlled'],
});

const context = await browser.newContext({
  userAgent: getRandomUA(),
});

// Stealth plugin automatically handles:
// - navigator.webdriver
// - chrome.runtime
// - WebGL fingerprint
// - Plugin injection
// - Permission API patches
// - ...and more
```

### Feature Parity Assessment

| Capability              | Go (undetected-chromedp) | JS (playwright-extra) |
| ----------------------- | ------------------------ | --------------------- |
| Automation flag removal | ✅                        | ✅                     |
| UA override             | ✅                        | ✅                     |
| WebGL spoofing          | ⚠️ Manual                 | ✅ Automatic           |
| Plugin injection        | ❌                        | ✅ Automatic           |
| Chrome runtime          | ⚠️ Manual                 | ✅ Automatic           |
| Permission API          | ❌                        | ✅ Automatic           |
| Source URL hiding       | ❌                        | ✅ Automatic           |

**Conclusion:** The JS implementation with playwright-extra actually provides **better** stealth capabilities than the current Go implementation.

---

*Document Version: 1.0.1*  
*Last Updated: 2026-02-03*
