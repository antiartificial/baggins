package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/antiartificial/baggins/internal/processor"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMediaProcessor is a mock implementation of the MediaProcessor
type MockMediaProcessor struct {
	mock.Mock
}

func (m MockMediaProcessor) DownloadMedia(url string) (*processor.Job, error) {
	args := m.Called(url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*processor.Job), args.Error(1)
}

func (m MockMediaProcessor) ExtractAudio(inputPath string, startTime, duration float64) (*processor.Job, error) {
	args := m.Called(inputPath, startTime, duration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*processor.Job), args.Error(1)
}

func (m MockMediaProcessor) ProcessYouTube(url string, startTime, duration float64) (*processor.Job, error) {
	args := m.Called(url, startTime, duration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*processor.Job), args.Error(1)
}

func (m MockMediaProcessor) GetJob(id string) *processor.Job {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*processor.Job)
}

func setupTestApp(proc processor.MediaProcessorInterface) *fiber.App {
	app := fiber.New()
	SetupRoutes(app, proc)
	return app
}

type testCase struct {
	name           string
	method         string
	path           string
	body           interface{}
	setupMock      func(*MockMediaProcessor)
	expectedStatus int
	expectedBody   map[string]interface{}
}

func TestAPIEndpoints(t *testing.T) {
	tests := []testCase{
		{
			name:   "Health Check",
			method: "GET",
			path:   "/api/health",
			setupMock: func(m *MockMediaProcessor) {
				// No mock setup needed for health check
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status": "ok",
			},
		},
		{
			name:   "Process Media - Success",
			method: "POST",
			path:   "/api/process",
			body: MediaRequest{
				URL:       "http://example.com/video.mp4",
				StartTime: 0,
				Duration: 60,
			},
			setupMock: func(m *MockMediaProcessor) {
				m.On("DownloadMedia", "http://example.com/video.mp4").Return(
					&processor.Job{
						ID:       "test-job-id",
						Status:   "downloaded",
						FilePath: "test-file-path",
					}, 
					nil,
				)
				m.On("ExtractAudio", "test-file-path", float64(0), float64(60)).Return(
					&processor.Job{
						ID:     "test-job-id",
						Status: "completed",
					},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"job_id": "test-job-id",
				"status": "completed",
			},
		},
		{
			name:   "Process Media - Invalid URL",
			method: "POST",
			path:   "/api/process",
			body: MediaRequest{
				URL: "",
			},
			setupMock: func(m *MockMediaProcessor) {
				// No mock setup needed for validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "URL is required",
			},
		},
		{
			name:   "Get Job Status - Success",
			method: "GET",
			path:   "/api/status/test-job-id",
			setupMock: func(m *MockMediaProcessor) {
				m.On("GetJob", "test-job-id").Return(
					&processor.Job{
						ID:     "test-job-id",
						Status: "completed",
					},
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"job_id": "test-job-id",
				"status": "completed",
				"error": nil,
				"filepath": "",
			},
		},
		{
			name:   "Get Job Status - Not Found",
			method: "GET",
			path:   "/api/status/non-existent",
			setupMock: func(m *MockMediaProcessor) {
				m.On("GetJob", "non-existent").Return(nil)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error": "Job not found",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockProc := new(MockMediaProcessor)
			tc.setupMock(mockProc)
			app := setupTestApp(*mockProc)

			var req *http.Request
			if tc.body != nil {
				jsonBody, _ := json.Marshal(tc.body)
				req = httptest.NewRequest(tc.method, tc.path, bytes.NewReader(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			var actualBody map[string]interface{}
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &actualBody)

			assert.Equal(t, tc.expectedBody, actualBody)
			mockProc.AssertExpectations(t)
		})
	}
}

func TestUploadEndpoint(t *testing.T) {
	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll("uploads", 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("uploads")

	tests := []struct {
		name           string
		fileContent    []byte
		filename       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Successful Upload",
			fileContent:    []byte("test file content"),
			filename:       "test.mp3",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "No File Provided",
			fileContent:    nil,
			filename:       "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "No file uploaded",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockProc := new(MockMediaProcessor)
			app := setupTestApp(*mockProc)

			var body bytes.Buffer
			writer := multipart.NewWriter(&body)

			if tc.fileContent != nil {
				part, err := writer.CreateFormFile("file", tc.filename)
				if err != nil {
					t.Fatal(err)
				}
				part.Write(tc.fileContent)
			}
			writer.Close()

			req := httptest.NewRequest("POST", "/api/upload", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)

			if tc.expectedError != "" {
				assert.Equal(t, tc.expectedError, result["error"])
			} else {
				assert.NotEmpty(t, result["job_id"])
				assert.Equal(t, "uploaded", result["status"])
			}
		})
	}
}
