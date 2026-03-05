# MathWizz Frontend Service
# React application with pixel art theme

# Build stage
FROM docker.io/library/node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Prefer IPv4 to avoid IPv6 connectivity issues in containers
ENV NODE_OPTIONS="--dns-result-order=ipv4first"

# Install dependencies (including dev dependencies needed for build)
RUN npm ci

# Copy source code
COPY public ./public
COPY src ./src

# Set API URL for build
ARG REACT_APP_API_URL=http://localhost:8080
ENV REACT_APP_API_URL=$REACT_APP_API_URL

# Build the application
RUN npm run build

# Production stage
FROM docker.io/library/nginx:alpine

# Copy custom nginx configuration
RUN echo 'server { \
    listen 80; \
    location / { \
        root /usr/share/nginx/html; \
        index index.html; \
        try_files $uri $uri/ /index.html; \
    } \
}' > /etc/nginx/conf.d/default.conf

# Copy built files from builder
COPY --from=builder /app/build /usr/share/nginx/html

# Expose HTTP port
EXPOSE 80

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost/ || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
