# WhatsApp Bridge - "The Hands" of GoClaw
FROM node:20-slim

# Install bridge system dependencies for native Baileys bindings
RUN apt-get update && apt-get install -y \
    ffmpeg \
    libvips \
    libvips-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY package*.json ./
RUN npm install --production

COPY . .

# Build TypeScript if necessary (or we run via ts-node in dev)
# For now, we'll expose for both JS/TS execution
EXPOSE 3000

# Persist auth_info via Docker volume for single-QR scan persistence
VOLUME [ "/app/auth_info" ]

CMD [ "npx", "ts-node", "src/index.ts" ]
