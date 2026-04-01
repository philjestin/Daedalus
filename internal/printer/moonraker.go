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

// MoonrakerClient implements Client for Moonraker API (Klipper).
type MoonrakerClient struct {
	printerID      uuid.UUID
	baseURL        string
	httpClient     *http.Client
	statusCallback func(*model.PrinterState)
	stopPolling    chan struct{}
}

// NewMoonrakerClient creates a new Moonraker client.
func NewMoonrakerClient(printerID uuid.UUID, baseURL string) *MoonrakerClient {
	return &MoonrakerClient{
		printerID: printerID,
		baseURL:   baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		stopPolling: make(chan struct{}),
	}
}

// Connect establishes connection and starts status polling.
func (c *MoonrakerClient) Connect() error {
	// Verify connection by getting server info
	_, err := c.doRequest("GET", "/server/info", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Moonraker: %w", err)
	}

	go c.pollStatus()
	return nil
}

// Disconnect stops polling.
func (c *MoonrakerClient) Disconnect() error {
	close(c.stopPolling)
	return nil
}

// GetStatus retrieves current printer status.
func (c *MoonrakerClient) GetStatus() (*model.PrinterState, error) {
	// Get printer status
	resp, err := c.doRequest("GET", "/printer/objects/query?print_stats&extruder&heater_bed", nil)
	if err != nil {
		// Connection failure means the printer is offline, not an application error
		offlineState := &model.PrinterState{PrinterID: c.printerID, Status: model.PrinterStatusOffline, UpdatedAt: time.Now()}
		return offlineState, nil //nolint:nilerr
	}

	state := c.parseState(resp)
	return state, nil
}

// StartJob uploads and starts printing a file.
func (c *MoonrakerClient) StartJob(filename string, filePath string) error {
	// Upload file
	if err := c.uploadFile(filePath); err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	// Start print
	_, err := c.doRequest("POST", "/printer/print/start?filename="+filename, nil)
	if err != nil {
		return fmt.Errorf("failed to start print: %w", err)
	}

	return nil
}

// PauseJob pauses the current print.
func (c *MoonrakerClient) PauseJob() error {
	_, err := c.doRequest("POST", "/printer/print/pause", nil)
	return err
}

// ResumeJob resumes a paused print.
func (c *MoonrakerClient) ResumeJob() error {
	_, err := c.doRequest("POST", "/printer/print/resume", nil)
	return err
}

// CancelJob cancels the current print.
func (c *MoonrakerClient) CancelJob() error {
	_, err := c.doRequest("POST", "/printer/print/cancel", nil)
	return err
}

// SetStatusCallback sets the callback for status updates.
func (c *MoonrakerClient) SetStatusCallback(cb func(*model.PrinterState)) {
	c.statusCallback = cb
}

// doRequest performs an HTTP request to the Moonraker API.
func (c *MoonrakerClient) doRequest(method string, path string, body []byte) ([]byte, error) { //nolint:unparam // body kept for future POST/PUT support
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

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

// uploadFile uploads a file to Moonraker.
func (c *MoonrakerClient) uploadFile(filePath string) error {
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

	req, err := http.NewRequest("POST", c.baseURL+"/server/files/upload", body)
	if err != nil {
		return err
	}

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
func (c *MoonrakerClient) pollStatus() {
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

// parseState converts Moonraker API response to PrinterState.
func (c *MoonrakerClient) parseState(resp []byte) *model.PrinterState {
	state := &model.PrinterState{
		PrinterID: c.printerID,
		Status:    model.PrinterStatusIdle,
		UpdatedAt: time.Now(),
	}

	var data struct {
		Result struct {
			Status struct {
				PrintStats struct {
					State      string  `json:"state"`
					Filename   string  `json:"filename"`
					TotalDuration float64 `json:"total_duration"`
					PrintDuration float64 `json:"print_duration"`
				} `json:"print_stats"`
				Extruder struct {
					Temperature float64 `json:"temperature"`
				} `json:"extruder"`
				HeaterBed struct {
					Temperature float64 `json:"temperature"`
				} `json:"heater_bed"`
			} `json:"status"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &data); err == nil {
		state.NozzleTemp = data.Result.Status.Extruder.Temperature
		state.BedTemp = data.Result.Status.HeaterBed.Temperature
		state.CurrentFile = data.Result.Status.PrintStats.Filename

		switch data.Result.Status.PrintStats.State {
		case "printing":
			state.Status = model.PrinterStatusPrinting
		case "paused":
			state.Status = model.PrinterStatusPaused
		case "error":
			state.Status = model.PrinterStatusError
		case "standby", "complete":
			state.Status = model.PrinterStatusIdle
		}
	}

	return state
}

