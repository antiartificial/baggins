FROM golang:1.21-alpine

WORKDIR /app

# Install system dependencies
RUN apk add --no-cache \
    ffmpeg \
    python3 \
    py3-pip \
    gcc \
    musl-dev

# Install yt-dlp
RUN pip3 install --no-cache-dir yt-dlp

# Copy Go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the application
RUN go build -o main ./cmd/server

EXPOSE 8080

CMD ["./main"]
