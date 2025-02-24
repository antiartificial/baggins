package api

import (
	"strings"
	"github.com/antiartificial/baggins/internal/processor"
	"github.com/gofiber/fiber/v2"
)

type MediaRequest struct {
	URL       string  `json:"url"`
	StartTime float64 `json:"start_time"`
	Duration  float64 `json:"duration"`
}

func SetupRoutes(app *fiber.App, proc processor.MediaProcessorInterface) {
	api := app.Group("/api")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Upload media file
	api.Post("/upload", func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "No file uploaded",
			})
		}

		// Save the file
		job := &processor.Job{
			ID:     file.Filename,
			Status: "processing",
		}

		if err := c.SaveFile(file, "uploads/"+file.Filename); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to save file",
			})
		}

		return c.JSON(fiber.Map{
			"job_id": job.ID,
			"status": "uploaded",
		})
	})

	// Process media from URL
	api.Post("/process", func(c *fiber.Ctx) error {
		var req MediaRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.URL == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "URL is required",
			})
		}

		var job *processor.Job
		var err error

		// Check if it's a YouTube URL
		if isYouTubeURL(req.URL) {
			job, err = proc.ProcessYouTube(req.URL, req.StartTime, req.Duration)
		} else {
			job, err = proc.DownloadMedia(req.URL)
			if err == nil && (req.StartTime > 0 || req.Duration > 0) {
				job, err = proc.ExtractAudio(job.FilePath, req.StartTime, req.Duration)
			}
		}

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"job_id": job.ID,
			"status": job.Status,
		})
	})

	// Get job status
	api.Get("/status/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		job := proc.GetJob(id)
		if job == nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Job not found",
			})
		}

		return c.JSON(fiber.Map{
			"job_id":   job.ID,
			"status":   job.Status,
			"filepath": job.FilePath,
			"error":    job.Error,
		})
	})

	// Download processed file
	api.Get("/download/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		job := proc.GetJob(id)
		if job == nil || job.FilePath == "" {
			return c.Status(404).JSON(fiber.Map{
				"error": "File not found",
			})
		}

		return c.SendFile(job.FilePath)
	})
}

func isYouTubeURL(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}
