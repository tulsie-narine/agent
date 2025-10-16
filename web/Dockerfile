# syntax=docker/dockerfile:1

ARG NODE_VERSION=22.13.1

# --- Builder stage ---
FROM node:${NODE_VERSION}-slim AS builder
WORKDIR /app

# Install dependencies
COPY --link package.json .
COPY --link package-lock.json* .
RUN --mount=type=cache,target=/root/.npm \
    npm ci || npm install

# Copy the entire application source (including tsconfig.json, next.config.js)
COPY --link . .

# Ensure .next cache is clean for fresh build
RUN rm -rf .next .cache 2>/dev/null || true

# Build the application (Next.js/TypeScript)
RUN npm run build

# Remove dev dependencies to reduce image size
RUN npm prune --production

# --- Production stage ---
FROM node:${NODE_VERSION}-slim AS final
WORKDIR /app

# Create a non-root user
RUN addgroup --system --gid 1001 appgroup && \
    adduser --system --uid 1001 --ingroup appgroup appuser

# Copy only the necessary files from builder
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/next.config.js ./next.config.js
COPY --from=builder /app/tailwind.config.js ./tailwind.config.js
COPY --from=builder /app/postcss.config.js ./postcss.config.js

# If you have any other static/config files needed at runtime, copy them here

ENV NODE_ENV=production
ENV NODE_OPTIONS="--max-old-space-size=4096"

USER appuser

EXPOSE 3000

CMD ["npm", "start"]
