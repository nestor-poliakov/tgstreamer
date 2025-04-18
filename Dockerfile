FROM golang:1.24-alpine3.21 AS builder

WORKDIR /app
RUN apk add --no-cache curl && \
    curl -L https://github.com/yt-dlp/yt-dlp/releases/download/2025.03.27/yt-dlp_linux -o yt-dlp && \
    chmod +x yt-dlp
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o tgstreamer ./main.go


FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/tgstreamer .
COPY --from=builder /app/yt-dlp /bin
EXPOSE 8000
CMD ["./tgstreamer"]
