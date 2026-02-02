# Cinder-JS (Bun + Playwright) Dockerfile - Planning Only
# Base image provides Chromium and Playwright dependencies.
FROM mcr.microsoft.com/playwright:v1.50.1-jammy AS base

# Install Bun
RUN apt-get update && apt-get install -y curl ca-certificates \
  && curl -fsSL https://bun.sh/install | bash \
  && ln -s /root/.bun/bin/bun /usr/local/bin/bun \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install Playwright browsers (ensure Chromium deps)
RUN npx -y playwright@1.50.1 install --with-deps chromium

# Copy app sources (placeholder for future implementation)
COPY . /app

EXPOSE 8080

# Start command: API + worker monolith (placeholder)
CMD ["bun", "run", "start"]
