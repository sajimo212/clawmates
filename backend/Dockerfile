FROM golang:1.25-alpine AS builder
WORKDIR /app

ARG SERVICE

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/service ./cmd/${SERVICE}

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /out/service /service
ENTRYPOINT ["/service"]
