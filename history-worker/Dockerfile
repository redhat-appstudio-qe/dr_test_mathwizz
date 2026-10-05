# MathWizz History-Worker Service
# Async worker that processes problem_solved events from NATS

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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o history-worker .

# Runtime stage
FROM docker.io/library/alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/history-worker .

# No exposed ports (worker doesn't listen on HTTP)

# Run the binary
CMD ["./history-worker"]
# konflux-test trigger run-d135ccb0-274f-447d-a88d-29a92edfc5de 1790839452
# konflux-test trigger run-dd44008f-7fd4-41ed-b1df-735ccbcaef2c 1791113681
# konflux-test trigger run-3ad07b03-14af-4029-8e95-eaf0878cfe23 1791116990
# konflux-test trigger run-53224a03-7815-4442-9b90-1af437f23c21 1791186327
# konflux-test trigger run-448d8a4b-a3fa-4d46-abab-6bc59a98e743 1791192593
