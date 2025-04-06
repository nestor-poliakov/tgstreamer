FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o tgstreamer ./main.go
RUN apk add --no-cache curl && \
    curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux -o yt-dlp && \
    chmod +x yt-dlp


FROM alpine:latest

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/tgstreamer .
COPY --from=builder /app/yt-dlp .
EXPOSE 8080
CMD ["./tgstreamer"]
