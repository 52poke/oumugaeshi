FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o oumugaeshi .

FROM alpine:3.18

# Install FFmpeg
RUN apk add --no-cache ffmpeg ca-certificates

WORKDIR /app

COPY --from=builder /app/oumugaeshi /app/oumugaeshi

EXPOSE 8080

ENV S3_BUCKET=mediawiki
ENV S3_ENDPOINT=http://minio:9000
ENV S3_REGION=us-east-1
ENV LISTEN_ADDR=:8080

ENTRYPOINT ["/app/oumugaeshi"]
