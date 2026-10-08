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
# konflux-test trigger run-d135ccb0-274f-447d-a88d-29a92edfc5de 1790839454
# konflux-test trigger run-dd44008f-7fd4-41ed-b1df-735ccbcaef2c 1791113683
# konflux-test trigger run-3ad07b03-14af-4029-8e95-eaf0878cfe23 1791116992
# konflux-test trigger run-53224a03-7815-4442-9b90-1af437f23c21 1791186329
# konflux-test trigger run-448d8a4b-a3fa-4d46-abab-6bc59a98e743 1791192595
# konflux-test trigger run-1874b9f4-0663-4b2f-8485-71a73afd6586 1791195857
# konflux-test trigger run-c3abda7c-0c72-4821-81de-ce220a7f583d 1791215979
# konflux-test trigger run-fe53789e-ee9c-4cc0-8d8f-456d670f2cf7 1791273430
# konflux-test trigger run-a5d71963-3898-4991-9bbd-ae6d294a2341 1791273569
# konflux-test trigger run-ec8dd773-9658-4720-8f7e-5cb3a02acce4 1791290240
# konflux-test trigger run-0ad9bfde-b6b9-4381-afb3-8c30cee05671 1791319032
# konflux-test trigger run-2069783a-9431-421d-9bd8-779d1e312102 1791377967
# konflux-test trigger run-a4c8351f-a9b7-4c12-8ffb-a33b32e0ebff 1791396694
# konflux-test trigger run-76793001-8e0a-419b-9035-31a548d0dcb6 1791446830
# konflux-test trigger run-06ec86e8-5805-4c98-a26c-baf9a5bd02da 1791451961
