FROM node:24-alpine AS build

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG PUBLIC_SITE_URL=http://localhost:8080
ENV PUBLIC_SITE_URL=$PUBLIC_SITE_URL
RUN PUBLIC_SITE_URL="${PUBLIC_SITE_URL:-http://localhost:8080}" npm run build && npm prune --omit=dev

FROM node:24-alpine AS runtime
ENV HOST=0.0.0.0
ENV PORT=4321
ENV NODE_ENV=production
WORKDIR /app
RUN addgroup -S adicara && adduser -S -G adicara adicara
COPY --from=build --chown=adicara:adicara /app/dist ./dist
COPY --from=build --chown=adicara:adicara /app/node_modules ./node_modules
COPY --from=build --chown=adicara:adicara /app/package.json ./package.json
USER adicara
EXPOSE 4321
CMD ["node", "./dist/server/entry.mjs"]
