# cinder-js Dockerfile

# =============================================================================
# Stage 1: Dependencies
# =============================================================================
FROM oven/bun:1-slim AS deps

WORKDIR /app

# Copy package files
COPY package.json bun.lock* ./

# Install production dependencies only
RUN bun install --frozen-lockfile --production

# =============================================================================
# Stage 2: Builder (if bundling needed)
# =============================================================================
FROM oven/bun:1-slim AS builder

WORKDIR /app

# Copy package files and install all deps (including dev)
COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile

# Copy source code
COPY . .

# Build/bundle if needed
# RUN bun run build

# =============================================================================
# Stage 3: Production Image with Playwright
# =============================================================================
# Using Microsoft's Playwright image which includes Chromium and all deps
FROM mcr.microsoft.com/playwright:v1.50.0-jammy AS production

# Set environment
ENV NODE_ENV=production
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1

# Install Bun in the Playwright image
RUN apt-get update && \
    apt-get install -y curl unzip && \
    curl -fsSL https://bun.sh/install | bash && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Add Bun to PATH
ENV BUN_INSTALL=/root/.bun
ENV PATH="$BUN_INSTALL/bin:$PATH"

WORKDIR /app

# Copy production dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules

# Copy source code (or built bundle from builder stage)
COPY --from=builder /app/src ./src
COPY --from=builder /app/package.json ./

# Create non-root user for security
RUN useradd -m -s /bin/bash appuser && \
    chown -R appuser:appuser /app

# Note: Can't run as non-root with Playwright in some configurations
# USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Expose port
EXPOSE 8080

# Start command
CMD ["bun", "run", "src/index.ts"]
