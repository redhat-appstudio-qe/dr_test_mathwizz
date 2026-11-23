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
# konflux-test trigger run-1874b9f4-0663-4b2f-8485-71a73afd6586 1791195855
# konflux-test trigger run-c3abda7c-0c72-4821-81de-ce220a7f583d 1791215977
# konflux-test trigger run-fe53789e-ee9c-4cc0-8d8f-456d670f2cf7 1791273428
# konflux-test trigger run-a5d71963-3898-4991-9bbd-ae6d294a2341 1791273567
# konflux-test trigger run-ec8dd773-9658-4720-8f7e-5cb3a02acce4 1791290238
# konflux-test trigger run-228186f2-149e-4d0c-a207-da082c865e00 1791315859
# konflux-test trigger run-0ad9bfde-b6b9-4381-afb3-8c30cee05671 1791319030
# konflux-test trigger run-2069783a-9431-421d-9bd8-779d1e312102 1791377963
# konflux-test trigger run-a4c8351f-a9b7-4c12-8ffb-a33b32e0ebff 1791396692
# konflux-test trigger run-76793001-8e0a-419b-9035-31a548d0dcb6 1791446828
# konflux-test trigger run-06ec86e8-5805-4c98-a26c-baf9a5bd02da 1791451960
# lkmabt trigger run-25e29bcd-1788-435f-9114-56bf14aff3ba 1794952155
# lkmabt trigger run-c49f29ac-8b26-4f17-a890-c5b032e82fc6 1794953876
# lkmabt trigger run-b80dae32-b57d-4d7f-8396-f23b98f8ff47 1794998002
# lkmabt trigger run-ac4243a9-6cb6-4a01-aa1d-a7cd260776dc 1795034961
# lkmabt trigger run-1d579c64-d40e-4eb8-bc06-11b49a9d46f3 1795036219
# lkmabt trigger run-621b2ac3-97e4-4d4e-8a25-eaaf7c026144 1795037770
# lkmabt trigger run-54c72588-d2b2-4283-9dae-fad4aacefa1b 1795038998
# lkmabt trigger run-e9c36f6d-164f-4a77-bb60-46ccb93ccc39 1795080658
# lkmabt trigger run-2712c392-7268-489b-b0dc-04b1395721cd 1795088021
# lkmabt trigger run-4c107c61-5749-4421-948f-144c25c14ab6 1795088554
# lkmabt trigger run-2e062fc9-a15a-455c-94bf-e61c5a536739 1795090681
# lkmabt trigger run-b3ad96b2-4679-4c5e-a5fb-9b1f21377a0d 1795091771
# lkmabt trigger run-cf6e7fef-184d-4596-970b-1074056eed93 1795092759
# lkmabt trigger run-082b325a-ef8d-4813-abb0-d8fc999993bc 1795103274
# lkmabt trigger run-4abd267a-dbc2-4ca3-9030-c4bcadcda8e9 1795117028
# lkmabt trigger run-d3e533a9-af72-402c-bed2-21df7a39208c 1795122103
# lkmabt trigger run-a3ae8b34-1423-4fea-9dba-cc9dccb8385c 1795125060
# lkmabt trigger run-1213bc43-5647-4607-a54d-bbe855dd3472 1795128120
# lkmabt trigger run-bd2bc4f3-7de6-471a-9e7f-1f5fd5a3be19 1795131579
# lkmabt trigger run-49c2e005-8e31-423c-8eb1-3ccff8823fbe 1795186881
# lkmabt trigger run-1828b89a-10a5-4776-b4cd-79df08f719a6 1795336865
# lkmabt trigger run-f7afcb90-c5ab-4434-9951-e962621729f1 1795338620
# lkmabt trigger run-94df92c9-e383-4187-b899-c764ed379641 1795340443
# lkmabt trigger run-d41d6d12-9d0b-475c-9fda-1b1409b59a39 1795350882
# lkmabt trigger run-1d09eecc-4e52-4d3b-9cf7-398693df7bce 1795351417
# lkmabt trigger run-003728d3-4efa-4058-9729-6b679e283cf9 1795427738
# lkmabt trigger run-e547a81c-313c-444c-9c74-d30844597e84 1795428732
# lkmabt trigger run-95927cf6-5a17-425a-a8e5-74c91e170ac5 1795429335
# lkmabt trigger run-34aec3c1-3ba3-4e80-abc3-97ffc94f332a 1795429999
# lkmabt trigger run-368956fd-8a27-4182-9b98-b356bb07a815 1795430809
# lkmabt trigger run-24bf4ad2-d88c-4995-aad1-518ad04156bb 1795431450
# lkmabt trigger run-20f2c9d2-1363-4331-94a4-cd7a601d75b7 1795431960
# lkmabt trigger run-f90ae6e6-d805-44ee-bf9d-00aaede2a5bc 1795432613
# lkmabt trigger run-991f5e09-340e-4058-b4a8-df4827b0c566 1795440844
# lkmabt trigger run-991f5e09-340e-4058-b4a8-df4827b0c566 1795440970
# lkmabt trigger run-d221d880-7ea9-491e-9e05-f44218a6612c 1795441361
# lkmabt trigger run-8daec00a-f79c-4c5f-b036-2f2777a2cbc3 1795441943
# lkmabt trigger run-807f5def-42bc-4723-b158-8970b131192a 1795442316
# lkmabt trigger run-96e30c9f-40bd-4d5b-bae2-7174618b815b 1795442855
# lkmabt trigger run-32cec5eb-8bf5-4b1f-b7af-f6b35ddc33c1 1795443352
# lkmabt trigger run-5e3ddd46-c385-4077-b842-644fa9696c77 1795454459
# lkmabt trigger run-5e3ddd46-c385-4077-b842-644fa9696c77 1795454590
