# syntax=docker/dockerfile:1

FROM golang:1.18-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w -X main.version=docker' -o /out/spanforge ./cmd/spanforge

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/spanforge /spanforge
ENTRYPOINT ["/spanforge"]
