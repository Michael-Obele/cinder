# Anti-Detection & Evasion Strategy

## Goals
- Match or exceed Cinder-Go’s undetected-chromedp behavior.
- Reduce bot detection signals while retaining stability.

## Strategy Overview
1. **Playwright-Extra + Stealth Plugin**
   - Use `playwright-extra` as a drop-in Playwright wrapper.
   - Apply `puppeteer-extra-plugin-stealth` to reduce fingerprint signals.
2. **Fingerprint Hardening**
   - Override WebGL vendor/renderer.
   - Mask `navigator.webdriver`.
   - Stabilize timezone/locale/viewport.
3. **Behavior Shaping**
   - User-agent rotation (desktop-focused).
   - Consistent screen sizes and device scale factors.

## Plugin-Based Stealth
- `playwright-extra` supports plugins and is compatible with `puppeteer-extra-plugin-stealth`.
- Stealth plugin supports WebGL vendor spoofing and other evasion tactics.

## Detection Testing
- **Primary**: https://bot.sannysoft.com
- **Secondary**: https://fingerprintjs.com/demo

## Risk Notes
- Stealth plugin is community-maintained and not guaranteed to pass advanced detections.
- Some sites still detect automation via behavioral patterns or CAPTCHAs.

## Validation Plan
- Run test matrix weekly against known detection pages.
- Track “stealth score” regression in CI (future).

