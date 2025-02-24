package processor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type MediaProcessorInterface interface {
	DownloadMedia(url string) (*Job, error)
	ExtractAudio(inputPath string, startTime, duration float64) (*Job, error)
	ProcessYouTube(url string, startTime, duration float64) (*Job, error)
	GetJob(id string) *Job
}

type MediaProcessor struct {
	Jobs       map[string]*Job
	workerPool *FFmpegWorkerPool
}

type Job struct {
	ID       string
	Status   string
	FilePath string
	Error    error
}

func NewMediaProcessor() *MediaProcessor {
	return &MediaProcessor{
		Jobs:       make(map[string]*Job),
		workerPool: NewFFmpegWorkerPool(5), // Limit to 5 concurrent FFmpeg processes
	}
}

func (p *MediaProcessor) DownloadMedia(url string) (*Job, error) {
	job := &Job{
		ID:     uuid.New().String(),
		Status: "downloading",
	}
	p.Jobs[job.ID] = job

	go func() {
		defer func() {
			if job.Error != nil {
				job.Status = "failed"
			}
		}()

		// Create unique filename
		ext := filepath.Ext(url)
		if ext == "" {
			ext = ".mp4" // default extension
		}
		filename := filepath.Join("uploads", fmt.Sprintf("%s%s", job.ID, ext))

		// Download file
		resp, err := http.Get(url)
		if err != nil {
			job.Error = err
			return
		}
		defer resp.Body.Close()

		file, err := os.Create(filename)
		if err != nil {
			job.Error = err
			return
		}
		defer file.Close()

		if _, err := io.Copy(file, resp.Body); err != nil {
			job.Error = err
			return
		}

		job.FilePath = filename
		job.Status = "completed"
	}()

	return job, nil
}

func (p *MediaProcessor) ExtractAudio(inputPath string, startTime, duration float64) (*Job, error) {
	job := &Job{
		ID:     uuid.New().String(),
		Status: "processing",
	}
	p.Jobs[job.ID] = job

	go func() {
		// Get a worker from the pool with a timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// Try to acquire a worker
		jobCtx, err := p.workerPool.AcquireWorker(ctx, job.ID)
		if err != nil {
			job.Error = fmt.Errorf("failed to acquire worker: %v", err)
			job.Status = "failed"
			return
		}
		defer p.workerPool.ReleaseWorker(job.ID)
		outputPath := filepath.Join("processed", fmt.Sprintf("%s.mp3", job.ID))

		// Prepare FFmpeg command
		cmd := exec.CommandContext(jobCtx, "ffmpeg",
			"-ss", fmt.Sprintf("%.2f", startTime),
			"-i", inputPath,
			"-t", fmt.Sprintf("%.2f", duration),
			"-q:a", "0",
			"-map", "a",
			outputPath,
		)

		if err := cmd.Run(); err != nil {
			job.Error = err
			job.Status = "failed"
			return
		}

		job.FilePath = outputPath
		job.Status = "completed"
	}()

	return job, nil
}

func (p *MediaProcessor) ProcessYouTube(url string, startTime, duration float64) (*Job, error) {
	job := &Job{
		ID:     uuid.New().String(),
		Status: "processing",
	}
	p.Jobs[job.ID] = job

	go func() {
		// Download with yt-dlp
		outputTemplate := filepath.Join("uploads", job.ID + ".%(ext)s")
		cmd := exec.Command("yt-dlp",
			"--format", "bestaudio",
			"--extract-audio",
			"--audio-format", "mp3",
			"-o", outputTemplate,
			url,
		)

		if err := cmd.Run(); err != nil {
			job.Error = err
			job.Status = "failed"
			return
		}

		// Find the downloaded file
		files, err := filepath.Glob(filepath.Join("uploads", job.ID + ".*"))
		if err != nil || len(files) == 0 {
			job.Error = fmt.Errorf("failed to find downloaded file")
			job.Status = "failed"
			return
		}

		downloadedFile := files[0]
		job.FilePath = downloadedFile

		// Extract the segment if needed
		if startTime > 0 || duration > 0 {
			extractJob, err := p.ExtractAudio(downloadedFile, startTime, duration)
			if err != nil {
				job.Error = err
				job.Status = "failed"
				return
			}

			// Wait for extraction to complete
			for extractJob.Status == "processing" {
				// Simple polling
			}

			if extractJob.Error != nil {
				job.Error = extractJob.Error
				job.Status = "failed"
				return
			}

			job.FilePath = extractJob.FilePath
		}

		job.Status = "completed"
	}()

	return job, nil
}

func (p *MediaProcessor) GetJob(id string) *Job {
	return p.Jobs[id]
}
