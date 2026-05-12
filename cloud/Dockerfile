FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /hero-cloud ./cmd/hero-cloud/

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /hero-cloud /usr/local/bin/hero-cloud
ENTRYPOINT ["hero-cloud"]
