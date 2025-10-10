FROM golang:1.25-alpine3.22 AS builder

WORKDIR /app
COPY . .
RUN go build -o tgstreamer ./main.go


FROM alpine:3.22 AS downloader

WORKDIR /app
RUN apk add --no-cache curl
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/download/2025.09.26/yt-dlp_linux -o yt-dlp && \
    chmod +x yt-dlp


FROM alpine:3.22
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=downloader /app/yt-dlp /bin
COPY --from=builder /app/tgstreamer .
CMD ["./tgstreamer"]
