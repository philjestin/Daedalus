package printer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

// OctoPrintClient implements Client for OctoPrint API.
type OctoPrintClient struct {
	printerID      uuid.UUID
	baseURL        string
	apiKey         string
	httpClient     *http.Client
	statusCallback func(*model.PrinterState)
	stopPolling    chan struct{}
}

// NewOctoPrintClient creates a new OctoPrint client.
func NewOctoPrintClient(printerID uuid.UUID, baseURL string, apiKey string) *OctoPrintClient {
	return &OctoPrintClient{
		printerID: printerID,
		baseURL:   baseURL,
		apiKey:    apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		stopPolling: make(chan struct{}),
	}
}

// Connect establishes connection and starts status polling.
func (c *OctoPrintClient) Connect() error {
	// Verify connection by getting version
	_, err := c.doRequest("GET", "/api/version", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to OctoPrint: %w", err)
	}

	// Start polling for status updates
	go c.pollStatus()

	return nil
}

// Disconnect stops polling and closes connection.
func (c *OctoPrintClient) Disconnect() error {
	close(c.stopPolling)
	return nil
}

// GetStatus retrieves current printer status.
func (c *OctoPrintClient) GetStatus() (*model.PrinterState, error) {
	// Get printer state
	printerResp, err := c.doRequest("GET", "/api/printer", nil)
	if err != nil {
		// Connection failure means the printer is offline, not an application error
		offlineState := &model.PrinterState{PrinterID: c.printerID, Status: model.PrinterStatusOffline, UpdatedAt: time.Now()}
		return offlineState, nil //nolint:nilerr
	}

	// Get job state
	jobResp, err := c.doRequest("GET", "/api/job", nil)
	if err != nil {
		jobResp = []byte("{}")
	}

	state := c.parseState(printerResp, jobResp)
	return state, nil
}

// StartJob uploads and starts printing a file.
func (c *OctoPrintClient) StartJob(filename string, filepath string) error {
	// Upload file
	if err := c.uploadFile(filepath); err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	// Start print
	selectReq := map[string]interface{}{
		"command": "select",
		"print":   true,
	}
	body, _ := json.Marshal(selectReq)

	_, err := c.doRequest("POST", "/api/files/local/"+filename, body)
	if err != nil {
		return fmt.Errorf("failed to start print: %w", err)
	}

	return nil
}

// PauseJob pauses the current print.
func (c *OctoPrintClient) PauseJob() error {
	req := map[string]string{"command": "pause", "action": "pause"}
	body, _ := json.Marshal(req)
	_, err := c.doRequest("POST", "/api/job", body)
	return err
}

// ResumeJob resumes a paused print.
func (c *OctoPrintClient) ResumeJob() error {
	req := map[string]string{"command": "pause", "action": "resume"}
	body, _ := json.Marshal(req)
	_, err := c.doRequest("POST", "/api/job", body)
	return err
}

// CancelJob cancels the current print.
func (c *OctoPrintClient) CancelJob() error {
	req := map[string]string{"command": "cancel"}
	body, _ := json.Marshal(req)
	_, err := c.doRequest("POST", "/api/job", body)
	return err
}

// SetStatusCallback sets the callback for status updates.
func (c *OctoPrintClient) SetStatusCallback(cb func(*model.PrinterState)) {
	c.statusCallback = cb
}

// doRequest performs an HTTP request to the OctoPrint API.
func (c *OctoPrintClient) doRequest(method string, path string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Api-Key", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// uploadFile uploads a file to OctoPrint.
func (c *OctoPrintClient) uploadFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return err
	}

	if _, err := io.Copy(part, file); err != nil {
		return err
	}

	writer.Close()

	req, err := http.NewRequest("POST", c.baseURL+"/api/files/local", body)
	if err != nil {
		return err
	}

	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("upload failed: %d", resp.StatusCode)
	}

	return nil
}

// pollStatus periodically polls for status updates.
func (c *OctoPrintClient) pollStatus() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopPolling:
			return
		case <-ticker.C:
			state, err := c.GetStatus()
			if err == nil && c.statusCallback != nil {
				c.statusCallback(state)
			}
		}
	}
}

// parseState converts OctoPrint API responses to PrinterState.
func (c *OctoPrintClient) parseState(printerResp []byte, jobResp []byte) *model.PrinterState {
	state := &model.PrinterState{
		PrinterID: c.printerID,
		Status:    model.PrinterStatusIdle,
		UpdatedAt: time.Now(),
	}

	// Parse printer response
	var printerData struct {
		State struct {
			Text  string `json:"text"`
			Flags struct {
				Printing bool `json:"printing"`
				Paused   bool `json:"paused"`
				Error    bool `json:"error"`
				Ready    bool `json:"ready"`
			} `json:"flags"`
		} `json:"state"`
		Temperature struct {
			Bed struct {
				Actual float64 `json:"actual"`
			} `json:"bed"`
			Tool0 struct {
				Actual float64 `json:"actual"`
			} `json:"tool0"`
		} `json:"temperature"`
	}

	if err := json.Unmarshal(printerResp, &printerData); err == nil {
		state.BedTemp = printerData.Temperature.Bed.Actual
		state.NozzleTemp = printerData.Temperature.Tool0.Actual

		if printerData.State.Flags.Error {
			state.Status = model.PrinterStatusError
		} else if printerData.State.Flags.Paused {
			state.Status = model.PrinterStatusPaused
		} else if printerData.State.Flags.Printing {
			state.Status = model.PrinterStatusPrinting
		} else if printerData.State.Flags.Ready {
			state.Status = model.PrinterStatusIdle
		}
	}

	// Parse job response
	var jobData struct {
		Job struct {
			File struct {
				Name string `json:"name"`
			} `json:"file"`
		} `json:"job"`
		Progress struct {
			Completion   float64 `json:"completion"`
			PrintTimeLeft int    `json:"printTimeLeft"`
		} `json:"progress"`
	}

	if err := json.Unmarshal(jobResp, &jobData); err == nil {
		state.CurrentFile = jobData.Job.File.Name
		state.Progress = jobData.Progress.Completion
		state.TimeLeft = jobData.Progress.PrintTimeLeft
	}

	return state
}

