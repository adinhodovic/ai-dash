FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" \
  -o ai-dash ./cmd/ai-dash

FROM alpine:3.20

RUN apk --no-cache add ca-certificates

RUN addgroup -g 1001 -S ai-dash && \
  adduser -u 1001 -S ai-dash -G ai-dash

WORKDIR /app
COPY --from=builder /app/ai-dash .
COPY --from=builder /app/sessions.sample.json .

USER ai-dash

CMD ["./ai-dash"]
