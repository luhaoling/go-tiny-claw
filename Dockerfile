FROM golang:1.26.11-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/claw ./cmd/claw

FROM alpine:3.24

RUN apk add --no-cache ca-certificates bash tzdata
COPY --from=builder /out/claw /usr/local/bin/claw

WORKDIR /workspace
EXPOSE 48080
ENTRYPOINT ["/usr/local/bin/claw"]
