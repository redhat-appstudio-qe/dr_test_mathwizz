# MathWizz Web-Server Service
# Go-based API server with database and message queue connections

# Build stage
FROM docker.io/library/golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o web-server .

# Runtime stage
FROM docker.io/library/alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/web-server .

# Expose HTTP port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./web-server"]
# konflux-test trigger run-d135ccb0-274f-447d-a88d-29a92edfc5de 1790839449
# konflux-test trigger run-dd44008f-7fd4-41ed-b1df-735ccbcaef2c 1791113679
# konflux-test trigger run-3ad07b03-14af-4029-8e95-eaf0878cfe23 1791116987
# konflux-test trigger run-53224a03-7815-4442-9b90-1af437f23c21 1791186326
# konflux-test trigger run-448d8a4b-a3fa-4d46-abab-6bc59a98e743 1791192592
# konflux-test trigger run-1874b9f4-0663-4b2f-8485-71a73afd6586 1791195854
# konflux-test trigger run-c3abda7c-0c72-4821-81de-ce220a7f583d 1791215975
# konflux-test trigger run-fe53789e-ee9c-4cc0-8d8f-456d670f2cf7 1791273426
# konflux-test trigger run-a5d71963-3898-4991-9bbd-ae6d294a2341 1791273565
