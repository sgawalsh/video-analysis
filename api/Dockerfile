FROM node:26-alpine@sha256:6bd9ab43c0ec0190ca7b2d193906cb8b7edf6e7f4afa43ddeccd6452710bffb5 AS base

RUN apk update && apk upgrade --no-cache
RUN npm install -g npm@latest

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --omit=dev

# ---------- Runtime ----------
FROM gcr.io/distroless/nodejs24:latest AS runtime

WORKDIR /app

# copy production node_modules
COPY --from=base /app/node_modules ./node_modules

# copy application source
COPY src ./src
COPY migrations ./migrations
COPY package.json ./

USER nonroot
CMD ["src/server.js"]

# ---------- Migration Image ----------
FROM base AS migrate
WORKDIR /app

COPY --from=base /app/node_modules ./node_modules
COPY src ./src
COPY migrations ./migrations
COPY package.json ./

CMD ["node", "migrations/cli.js"]