# Baggins Media Service

A containerized Go service for processing media files and extracting audio segments from videos. Built with Fiber and supports YouTube video processing via yt-dlp.

## Features

- Upload media files directly
- Process media from URLs (including YouTube videos)
- Extract audio segments from videos with custom start time and duration
- Asynchronous processing with job status tracking
- RESTful API endpoints

## API Endpoints

### Health Check
```
GET /api/health
```

### Upload Media File
```
POST /api/upload
Content-Type: multipart/form-data
Form field: file
```

### Process Media from URL
```
POST /api/process
Content-Type: application/json

{
    "url": "https://example.com/video.mp4",
    "start_time": 10.5,    // Optional: Start time in seconds
    "duration": 30.0       // Optional: Duration in seconds
}
```

### Get Job Status
```
GET /api/status/:job_id
```

### Download Processed File
```
GET /api/download/:job_id
```

## Building and Running

1. Build the Docker image:
```bash
docker build -t baggins .
```

2. Run the container:
```bash
docker run -p 8080:8080 baggins
```

## Dependencies

- Go 1.21+
- FFmpeg
- yt-dlp
- Python 3

All dependencies are included in the Docker image.

## Environment Variables

- `PORT`: Server port (default: 8080)

## Storage

The service uses two main directories for file storage:
- `uploads/`: For storing uploaded and downloaded media files
- `processed/`: For storing processed audio segments
