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
# konflux-test trigger run-ec8dd773-9658-4720-8f7e-5cb3a02acce4 1791290236
# konflux-test trigger run-228186f2-149e-4d0c-a207-da082c865e00 1791315856
# konflux-test trigger run-0ad9bfde-b6b9-4381-afb3-8c30cee05671 1791319028
# konflux-test trigger run-2069783a-9431-421d-9bd8-779d1e312102 1791377962
# konflux-test trigger run-a4c8351f-a9b7-4c12-8ffb-a33b32e0ebff 1791396691
# konflux-test trigger run-76793001-8e0a-419b-9035-31a548d0dcb6 1791446826
# konflux-test trigger run-06ec86e8-5805-4c98-a26c-baf9a5bd02da 1791451958
# lkmabt trigger run-25e29bcd-1788-435f-9114-56bf14aff3ba 1794952153
# lkmabt trigger run-c49f29ac-8b26-4f17-a890-c5b032e82fc6 1794953873
# lkmabt trigger run-b80dae32-b57d-4d7f-8396-f23b98f8ff47 1794997999
# lkmabt trigger run-ac4243a9-6cb6-4a01-aa1d-a7cd260776dc 1795034959
# lkmabt trigger run-1d579c64-d40e-4eb8-bc06-11b49a9d46f3 1795036217
# lkmabt trigger run-621b2ac3-97e4-4d4e-8a25-eaaf7c026144 1795037768
# lkmabt trigger run-54c72588-d2b2-4283-9dae-fad4aacefa1b 1795038996
# lkmabt trigger run-e9c36f6d-164f-4a77-bb60-46ccb93ccc39 1795080656
# lkmabt trigger run-2712c392-7268-489b-b0dc-04b1395721cd 1795088019
# lkmabt trigger run-4c107c61-5749-4421-948f-144c25c14ab6 1795088552
# lkmabt trigger run-2e062fc9-a15a-455c-94bf-e61c5a536739 1795090680
