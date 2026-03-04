package printer

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/jlaffaye/ftp"
)

// BambuClient implements Client for Bambu Lab printers via LAN or Cloud MQTT.
// Bambu printers use MQTT over TLS on port 8883 for status and control.
// File uploads are done via FTPS on port 990 (LAN mode only).
type BambuClient struct {
	printerID      uuid.UUID
	host           string
	accessCode     string
	serialNumber   string
	statusCallback func(*model.PrinterState)

	// Cloud MQTT mode fields
	cloudMode    bool
	mqttUsername string // "u_{uid}" for cloud, "bblp" for LAN

	mu         sync.RWMutex
	mqttClient mqtt.Client
	connected  bool
	lastState  *model.PrinterState
	stopChan   chan struct{}
}

// BambuMQTTPort is the default MQTT port for Bambu printers.
const BambuMQTTPort = 8883

// BambuFTPSPort is the default FTPS port for Bambu printers.
const BambuFTPSPort = 990

// NewBambuClient creates a new Bambu LAN client.
// The host should be the IP address or hostname of the printer.
// If host is a full URI (http://IP:PORT), the hostname is extracted.
// The accessCode is the LAN access code from the printer's settings.
// The serialNumber is needed for MQTT topic subscription.
func NewBambuClient(printerID uuid.UUID, host string, accessCode string, serialNumber string) *BambuClient {
	// If host is a full URI, extract just the hostname
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		if u, err := url.Parse(host); err == nil {
			host = u.Hostname()
		}
	}

	serial := serialNumber
	if serial == "" {
		serial = "unknown"
	}

	return &BambuClient{
		printerID:    printerID,
		host:         host,
		accessCode:   accessCode,
		serialNumber: serial,
		stopChan:     make(chan struct{}),
		lastState: &model.PrinterState{
			PrinterID: printerID,
			Status:    model.PrinterStatusOffline,
			UpdatedAt: time.Now(),
		},
	}
}

// NewBambuCloudClient creates a Bambu client that connects via the cloud MQTT broker.
// mqttUsername is "u_{uid}" from the Bambu account preferences.
// authToken is the access token from Bambu Cloud login (used as MQTT password).
func NewBambuCloudClient(printerID uuid.UUID, serial, mqttUsername, authToken string) *BambuClient {
	return &BambuClient{
		printerID:    printerID,
		host:         "us.mqtt.bambulab.com",
		accessCode:   authToken,
		serialNumber: serial,
		cloudMode:    true,
		mqttUsername:  mqttUsername,
		stopChan:     make(chan struct{}),
		lastState: &model.PrinterState{
			PrinterID: printerID,
			Status:    model.PrinterStatusOffline,
			UpdatedAt: time.Now(),
		},
	}
}

// SetSerialNumber sets the printer's serial number for MQTT topics.
func (c *BambuClient) SetSerialNumber(serial string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.serialNumber = serial
}

// Connect establishes MQTT connection to Bambu printer.
// Connection pattern follows the bambulabs_api community SDK.
func (c *BambuClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected && c.mqttClient != nil && c.mqttClient.IsConnected() {
		return nil
	}

	broker := fmt.Sprintf("ssl://%s:%d", c.host, BambuMQTTPort)

	// Determine MQTT credentials based on mode
	username := "bblp"
	if c.cloudMode && c.mqttUsername != "" {
		username = c.mqttUsername
	}

	// TLS config differs between local and cloud mode
	var tlsConfig *tls.Config
	if c.cloudMode {
		// Cloud broker has a valid certificate — use system CA pool
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	} else {
		// Local printer uses self-signed cert
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	}

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(fmt.Sprintf("printfarm_%s", c.printerID.String()[:8])).
		SetUsername(username).
		SetPassword(c.accessCode).
		SetTLSConfig(tlsConfig).
		SetAutoReconnect(true).
		SetConnectTimeout(10 * time.Second).
		SetWriteTimeout(10 * time.Second).
		SetKeepAlive(30 * time.Second)

	// Use DefaultPublishHandler for all incoming messages (matches community SDK pattern)
	opts.SetDefaultPublishHandler(c.handleMessage)
	opts.SetOnConnectHandler(c.onConnect)
	opts.SetConnectionLostHandler(c.onConnectionLost)
	opts.SetReconnectingHandler(c.onReconnecting)

	c.mqttClient = mqtt.NewClient(opts)

	mode := "LAN"
	if c.cloudMode {
		mode = "cloud"
	}
	slog.Info("connecting to Bambu printer",
		"mode", mode,
		"host", c.host,
		"printer_id", c.printerID,
		"serial", c.serialNumber,
		"broker", broker,
		"username", username,
	)
	token := c.mqttClient.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("MQTT connection failed: %w", token.Error())
	}

	c.connected = true
	slog.Info("connected to Bambu printer", "mode", mode, "host", c.host, "printer_id", c.printerID)

	return nil
}

// onConnect is called when MQTT connection is established.
func (c *BambuClient) onConnect(client mqtt.Client) {
	slog.Info("Bambu MQTT connected",
		"printer_id", c.printerID,
		"host", c.host,
		"serial", c.serialNumber,
		"has_access_code", c.accessCode != "",
		"access_code_len", len(c.accessCode),
	)

	// Subscribe to report topic — matches community SDK pattern:
	// QoS 0, nil handler (messages go through DefaultPublishHandler)
	topic := fmt.Sprintf("device/%s/report", c.serialNumber)
	token := client.Subscribe(topic, 0, nil)
	if token.Wait() && token.Error() != nil {
		slog.Error("failed to subscribe to Bambu report topic",
			"error", token.Error(),
			"topic", topic,
			"serial", c.serialNumber,
		)
		return
	}
	slog.Info("subscribed to Bambu report topic", "topic", topic)

	// Request initial status
	c.requestPushAll()
}

// onConnectionLost is called when MQTT connection is lost.
func (c *BambuClient) onConnectionLost(client mqtt.Client, err error) {
	slog.Warn("Bambu MQTT connection lost", "printer_id", c.printerID, "error", err)

	c.mu.Lock()
	c.connected = false
	c.lastState.Status = model.PrinterStatusOffline
	c.lastState.UpdatedAt = time.Now()
	state := *c.lastState
	c.mu.Unlock()

	if c.statusCallback != nil {
		c.statusCallback(&state)
	}
}

// onReconnecting is called when MQTT client is attempting to reconnect.
func (c *BambuClient) onReconnecting(client mqtt.Client, opts *mqtt.ClientOptions) {
	slog.Info("Bambu MQTT reconnecting",
		"printer_id", c.printerID,
		"host", c.host,
		"serial", c.serialNumber,
		"has_access_code", c.accessCode != "",
		"access_code_len", len(c.accessCode),
	)
}

// handleMessage processes incoming MQTT messages from the printer.
func (c *BambuClient) handleMessage(client mqtt.Client, msg mqtt.Message) {
	var payload BambuMessage
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		slog.Debug("failed to parse Bambu message", "error", err, "payload", string(msg.Payload()))
		return
	}

	// Check for print status in the message
	if payload.Print != nil {
		c.mu.Lock()
		state := c.mergePrintStatus(payload.Print)
		c.lastState = state
		c.mu.Unlock()

		if c.statusCallback != nil {
			c.statusCallback(state)
		}
	}
}

// requestPushAll requests the printer to send all current status.
func (c *BambuClient) requestPushAll() {
	cmd := BambuCommand{
		Pushing: &BambuPushingCmd{
			Sequence: strconv.FormatInt(time.Now().UnixMilli(), 10),
			Command:  "pushall",
		},
	}
	c.sendCommand(cmd) //nolint:errcheck // fire-and-forget status refresh
}

// sendCommand sends a command to the printer via MQTT.
func (c *BambuClient) sendCommand(cmd BambuCommand) error {
	c.mu.RLock()
	if !c.connected || c.mqttClient == nil {
		c.mu.RUnlock()
		return fmt.Errorf("not connected")
	}
	client := c.mqttClient
	serial := c.serialNumber
	c.mu.RUnlock()

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	topic := fmt.Sprintf("device/%s/request", serial)
	token := client.Publish(topic, 1, false, data)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("command publish timeout")
	}
	if token.Error() != nil {
		return fmt.Errorf("command publish failed: %w", token.Error())
	}

	slog.Debug("sent Bambu command", "topic", topic, "command", string(data))
	return nil
}

// Disconnect closes the MQTT connection.
func (c *BambuClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopChan != nil {
		close(c.stopChan)
		c.stopChan = nil
	}

	if c.mqttClient != nil && c.mqttClient.IsConnected() {
		c.mqttClient.Disconnect(1000)
	}
	c.connected = false

	slog.Info("disconnected from Bambu printer", "printer_id", c.printerID)
	return nil
}

// GetStatus retrieves current printer status.
func (c *BambuClient) GetStatus() (*model.PrinterState, error) {
	c.mu.RLock()
	state := c.lastState
	c.mu.RUnlock()

	if state == nil {
		return &model.PrinterState{
			PrinterID: c.printerID,
			Status:    model.PrinterStatusOffline,
			UpdatedAt: time.Now(),
		}, nil
	}

	return state, nil
}

// StartJob sends a print job to the Bambu printer.
// The file is uploaded via FTPS, then a print command is sent via MQTT.
func (c *BambuClient) StartJob(filename string, filepath string) error {
	// First upload the file via FTPS
	remotePath, err := c.uploadFile(filepath, filename)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	// Send print command via MQTT
	cmd := BambuCommand{
		Print: &BambuPrintCmd{
			Sequence:    strconv.FormatInt(time.Now().UnixMilli(), 10),
			Command:     "project_file",
			ProjectID:   "0",
			ProfileID:   "0",
			TaskID:      "0",
			Subtask:     "0",
			SubtaskName: filename,
			URL:         fmt.Sprintf("ftp://%s", remotePath),
			Filename:    filename,
			Timelapse:   false,
			BedLeveling: true,
			FlowCali:    false,
			Vibration:   false,
			AMS: &BambuAMSCmd{
				UseAMS:        true,
				PrintSpeed:    "standard",
				LayerInspect:  false,
				LabelVersion:  "",
				AMSMappingInfo: []int{},
			},
		},
	}

	if err := c.sendCommand(cmd); err != nil {
		return fmt.Errorf("failed to send print command: %w", err)
	}

	slog.Info("started print job on Bambu printer", "printer_id", c.printerID, "file", filename)
	return nil
}

// uploadFile uploads a file to the Bambu printer via FTPS.
func (c *BambuClient) uploadFile(localPath string, remoteName string) (string, error) {
	// Open local file
	file, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Connect to FTP with TLS
	ftpAddr := fmt.Sprintf("%s:%d", c.host, BambuFTPSPort)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	}

	conn, err := ftp.Dial(ftpAddr,
		ftp.DialWithTimeout(30*time.Second),
		ftp.DialWithExplicitTLS(tlsConfig),
	)
	if err != nil {
		// Try implicit TLS
		conn, err = ftp.Dial(ftpAddr,
			ftp.DialWithTimeout(30*time.Second),
			ftp.DialWithTLS(tlsConfig),
		)
		if err != nil {
			return "", fmt.Errorf("failed to connect to FTP: %w", err)
		}
	}
	defer conn.Quit()

	// Login - Bambu uses "bblp" as username and access code as password
	if err := conn.Login("bblp", c.accessCode); err != nil {
		return "", fmt.Errorf("FTP login failed: %w", err)
	}

	// Determine remote directory
	// Bambu printers typically store files in /cache or root
	remoteDir := "/cache"
	if err := conn.ChangeDir(remoteDir); err != nil {
		remoteDir = "/"
		conn.ChangeDir(remoteDir) //nolint:errcheck // fallback to root dir
	}

	// Upload file
	remotePath := filepath.Join(remoteDir, remoteName)
	if err := conn.Stor(remoteName, file); err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	slog.Info("uploaded file to Bambu printer", "remote_path", remotePath)
	return remotePath, nil
}

// PauseJob pauses the current print via MQTT command.
func (c *BambuClient) PauseJob() error {
	cmd := BambuCommand{
		Print: &BambuPrintCmd{
			Sequence: strconv.FormatInt(time.Now().UnixMilli(), 10),
			Command:  "pause",
		},
	}

	if err := c.sendCommand(cmd); err != nil {
		return fmt.Errorf("failed to pause: %w", err)
	}

	slog.Info("paused print on Bambu printer", "printer_id", c.printerID)
	return nil
}

// ResumeJob resumes a paused print via MQTT command.
func (c *BambuClient) ResumeJob() error {
	cmd := BambuCommand{
		Print: &BambuPrintCmd{
			Sequence: strconv.FormatInt(time.Now().UnixMilli(), 10),
			Command:  "resume",
		},
	}

	if err := c.sendCommand(cmd); err != nil {
		return fmt.Errorf("failed to resume: %w", err)
	}

	slog.Info("resumed print on Bambu printer", "printer_id", c.printerID)
	return nil
}

// CancelJob cancels the current print via MQTT command.
func (c *BambuClient) CancelJob() error {
	cmd := BambuCommand{
		Print: &BambuPrintCmd{
			Sequence: strconv.FormatInt(time.Now().UnixMilli(), 10),
			Command:  "stop",
		},
	}

	if err := c.sendCommand(cmd); err != nil {
		return fmt.Errorf("failed to cancel: %w", err)
	}

	slog.Info("cancelled print on Bambu printer", "printer_id", c.printerID)
	return nil
}

// SetStatusCallback sets the callback for status updates.
func (c *BambuClient) SetStatusCallback(cb func(*model.PrinterState)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statusCallback = cb
}

// GetLiveView returns a reader for the printer's live camera feed.
// Bambu cameras stream on port 6000 (RTSP) or via snapshot URL.
func (c *BambuClient) GetLiveView() (io.ReadCloser, error) {
	// Bambu cameras are typically accessed via RTSP or HTTP snapshot
	// This would require additional implementation
	return nil, fmt.Errorf("live view not yet implemented")
}

// mergePrintStatus merges incoming Bambu print data into the existing state.
// Bambu sends partial MQTT updates — a message may contain only temperatures,
// only progress, or only gcode_state. We must preserve fields from prior
// messages rather than resetting them to zero values.
// Caller must hold c.mu.
// bambuFanToPercent converts Bambu's 0-15 fan speed scale to 0-100%.
func bambuFanToPercent(raw int) int {
	if raw <= 0 {
		return 0
	}
	if raw >= 15 {
		return 100
	}
	return (raw * 100) / 15
}

func (c *BambuClient) mergePrintStatus(print *BambuPrintStatus) *model.PrinterState {
	// Start from a copy of the existing state
	state := &model.PrinterState{
		PrinterID:         c.printerID,
		Status:            c.lastState.Status,
		Progress:          c.lastState.Progress,
		CurrentFile:       c.lastState.CurrentFile,
		TimeLeft:          c.lastState.TimeLeft,
		BedTemp:           c.lastState.BedTemp,
		NozzleTemp:        c.lastState.NozzleTemp,
		AMS:               c.lastState.AMS,
		UpdatedAt:         time.Now(),
		BedTargetTemp:     c.lastState.BedTargetTemp,
		NozzleTargetTemp:  c.lastState.NozzleTargetTemp,
		ChamberTemp:       c.lastState.ChamberTemp,
		LayerNum:          c.lastState.LayerNum,
		TotalLayerNum:     c.lastState.TotalLayerNum,
		CoolingFanSpeed:   c.lastState.CoolingFanSpeed,
		AuxFanSpeed:       c.lastState.AuxFanSpeed,
		ChamberFanSpeed:   c.lastState.ChamberFanSpeed,
		HeatbreakFanSpeed: c.lastState.HeatbreakFanSpeed,
		SpeedPercent:      c.lastState.SpeedPercent,
		SpeedLevel:        c.lastState.SpeedLevel,
		PrintRealSpeed:    c.lastState.PrintRealSpeed,
		WiFiSignal:        c.lastState.WiFiSignal,
		NozzleDiameter:    c.lastState.NozzleDiameter,
		NozzleType:        c.lastState.NozzleType,
		HMSErrors:         c.lastState.HMSErrors,
		Lights:            c.lastState.Lights,
		GcodeStartTime:    c.lastState.GcodeStartTime,
		SubtaskID:         c.lastState.SubtaskID,
		TaskID:            c.lastState.TaskID,
		PrintType:         c.lastState.PrintType,
	}

	// Only update status when gcode_state is actually present
	if print.GcodeState != "" {
		switch print.GcodeState {
		case "IDLE":
			state.Status = model.PrinterStatusIdle
		case "RUNNING", "PREPARE":
			state.Status = model.PrinterStatusPrinting
		case "PAUSE":
			state.Status = model.PrinterStatusPaused
		case "FINISH":
			state.Status = model.PrinterStatusIdle
		case "FAILED":
			state.Status = model.PrinterStatusError
		}
	} else if print.StgCur > 0 && print.StgCur < 255 {
		// No gcode_state but stg_cur indicates activity
		state.Status = model.PrinterStatusPrinting
	}

	// Progress — MCPercent 0 is valid (just started), but we only
	// overwrite when the field was actually in the JSON. Since Go
	// can't distinguish 0 from absent with plain int, we update
	// whenever gcode_state is present (full status push) or
	// MCPercent > 0.
	if print.MCPercent > 0 || print.GcodeState != "" {
		state.Progress = float64(print.MCPercent)
	}

	// Current file
	if print.SubtaskName != "" {
		state.CurrentFile = print.SubtaskName
	} else if print.GcodeFile != "" {
		state.CurrentFile = print.GcodeFile
	}

	// Time remaining (in minutes from Bambu)
	if print.MCRemainingTime > 0 {
		state.TimeLeft = print.MCRemainingTime * 60 // Convert to seconds
	} else if print.GcodeState == "IDLE" || print.GcodeState == "FINISH" {
		state.TimeLeft = 0
	}

	// Temperatures — always update when present (non-zero)
	if print.BedTargetTemper > 0 || print.BedTemper > 0 {
		state.BedTemp = print.BedTemper
	}
	if print.NozzleTargetTemper > 0 || print.NozzleTemper > 0 {
		state.NozzleTemp = print.NozzleTemper
	}

	// Target temperatures
	if print.BedTargetTemper > 0 || print.GcodeState != "" {
		state.BedTargetTemp = print.BedTargetTemper
	}
	if print.NozzleTargetTemper > 0 || print.GcodeState != "" {
		state.NozzleTargetTemp = print.NozzleTargetTemper
	}
	if print.ChamberTemper > 0 || print.GcodeState != "" {
		state.ChamberTemp = print.ChamberTemper
	}

	// Layers
	if print.LayerNum > 0 || print.GcodeState != "" {
		state.LayerNum = print.LayerNum
	}
	if print.TotalLayerNum > 0 || print.GcodeState != "" {
		state.TotalLayerNum = print.TotalLayerNum
	}

	// Fan speeds (Bambu uses 0-15 scale)
	if print.CoolingFanSpeed > 0 || print.GcodeState != "" {
		state.CoolingFanSpeed = bambuFanToPercent(print.CoolingFanSpeed)
	}
	if print.BigFan1Speed > 0 || print.GcodeState != "" {
		state.AuxFanSpeed = bambuFanToPercent(print.BigFan1Speed)
	}
	if print.BigFan2Speed > 0 || print.GcodeState != "" {
		state.ChamberFanSpeed = bambuFanToPercent(print.BigFan2Speed)
	}
	if print.HeatbreakFanSpeed > 0 || print.GcodeState != "" {
		state.HeatbreakFanSpeed = bambuFanToPercent(print.HeatbreakFanSpeed)
	}

	// Speed
	if print.SpeedMag > 0 || print.GcodeState != "" {
		state.SpeedPercent = print.SpeedMag
	}
	if print.SpeedLevel > 0 || print.GcodeState != "" {
		state.SpeedLevel = print.SpeedLevel
	}
	if print.PrintRealSpeed > 0 {
		state.PrintRealSpeed = print.PrintRealSpeed
	}

	// Network
	if print.WiFiSignal != "" {
		state.WiFiSignal = print.WiFiSignal
	}

	// Nozzle info
	if print.NozzleDiameter != "" {
		state.NozzleDiameter = print.NozzleDiameter
	}
	if print.NozzleType != "" {
		state.NozzleType = print.NozzleType
	}

	// HMS errors
	if print.HMS != nil {
		state.HMSErrors = make([]model.HMSError, len(print.HMS))
		for i, h := range print.HMS {
			state.HMSErrors[i] = model.HMSError{
				Attr:     h.Attr,
				Code:     h.Code,
				Module:   h.Module,
				Severity: h.Severity,
			}
		}
	}

	// Lights
	if print.LightsReport != nil {
		state.Lights = make([]model.LightState, len(print.LightsReport))
		for i, l := range print.LightsReport {
			state.Lights[i] = model.LightState{
				Node: l.Node,
				Mode: l.Mode,
			}
		}
	}

	// Timing / Job IDs
	if print.PrintStartTime != "" {
		state.GcodeStartTime = print.PrintStartTime
	}
	if print.SubtaskID != "" {
		state.SubtaskID = print.SubtaskID
	}
	if print.TaskID != "" {
		state.TaskID = print.TaskID
	}
	if print.PrintType != "" {
		state.PrintType = print.PrintType
	}

	// Clear layer/time when idle or finished
	if print.GcodeState == "IDLE" || print.GcodeState == "FINISH" {
		state.LayerNum = 0
		state.TotalLayerNum = 0
		state.TimeLeft = 0
	}

	// Parse AMS state
	if print.AMS != nil {
		state.AMS = c.parseAMSState(print.AMS, print.VTTray)
	}

	return state
}

// parseAMSState converts Bambu AMS data to model.AMSState.
func (c *BambuClient) parseAMSState(ams *BambuAMS, vtTray *BambuVTTray) *model.AMSState {
	state := &model.AMSState{
		CurrentTray: ams.TrayNow,
	}

	// Parse AMS units
	for _, unit := range ams.Units {
		unitID, _ := strconv.Atoi(unit.ID)
		humidity, _ := strconv.Atoi(unit.Humidity)
		temp, _ := strconv.ParseFloat(unit.Temp, 64)

		amsUnit := model.AMSUnit{
			ID:       unitID,
			Humidity: humidity,
			Temp:     temp,
		}

		// Parse trays in unit
		for _, tray := range unit.Trays {
			trayID, _ := strconv.Atoi(tray.ID)
			nozzleMin, _ := strconv.Atoi(tray.NozzleTempMin)
			nozzleMax, _ := strconv.Atoi(tray.NozzleTempMax)
			bedTemp, _ := strconv.Atoi(tray.BedTemp)

			amsTray := model.AMSTray{
				ID:            trayID,
				MaterialType:  tray.TrayType,
				Color:         tray.TraySubBrands,
				ColorHex:      tray.TrayColor,
				Remain:        tray.Remain,
				TagUID:        tray.TagUID,
				Brand:         tray.TraySubBrands,
				NozzleTempMin: nozzleMin,
				NozzleTempMax: nozzleMax,
				BedTemp:       bedTemp,
				Empty:         tray.TrayType == "" || tray.Remain == 0,
			}
			amsUnit.Trays = append(amsUnit.Trays, amsTray)
		}

		state.Units = append(state.Units, amsUnit)
	}

	// Parse external spool (VT tray)
	if vtTray != nil && vtTray.TrayType != "" {
		nozzleMin, _ := strconv.Atoi(vtTray.NozzleTempMin)
		nozzleMax, _ := strconv.Atoi(vtTray.NozzleTempMax)
		bedTemp, _ := strconv.Atoi(vtTray.BedTemp)

		state.ExternalSpool = &model.AMSTray{
			ID:            255, // External spool uses ID 255
			MaterialType:  vtTray.TrayType,
			Color:         vtTray.TraySubBrands,
			ColorHex:      vtTray.TrayColor,
			Remain:        vtTray.Remain,
			Brand:         vtTray.TraySubBrands,
			NozzleTempMin: nozzleMin,
			NozzleTempMax: nozzleMax,
			BedTemp:       bedTemp,
			Empty:         vtTray.TrayType == "" || vtTray.Remain == 0,
		}
	}

	return state
}

// BambuMessage represents the top-level structure of Bambu MQTT messages.
type BambuMessage struct {
	Print   *BambuPrintStatus `json:"print,omitempty"`
	Info    *BambuInfo        `json:"info,omitempty"`
	System  *BambuSystem      `json:"system,omitempty"`
	Pushing *BambuPushing     `json:"pushing,omitempty"`
}

// BambuPrintStatus contains print job status information.
type BambuPrintStatus struct {
	// Print state
	GcodeState    string `json:"gcode_state"`
	StgCur        int    `json:"stg_cur"`
	PrintError    int    `json:"print_error"`
	PrintType     string `json:"print_type"`
	SubtaskName   string `json:"subtask_name"`
	GcodeFile     string `json:"gcode_file"`
	SubtaskID     string `json:"subtask_id"`
	TaskID        string `json:"task_id"`

	// Progress
	MCPercent       int `json:"mc_percent"`
	MCRemainingTime int `json:"mc_remaining_time"`
	LayerNum        int `json:"layer_num"`
	TotalLayerNum   int `json:"total_layer_num"`

	// Temperatures
	BedTemper         float64 `json:"bed_temper"`
	BedTargetTemper   float64 `json:"bed_target_temper"`
	NozzleTemper      float64 `json:"nozzle_temper"`
	NozzleTargetTemper float64 `json:"nozzle_target_temper"`
	ChamberTemper     float64 `json:"chamber_temper"`
	NozzleDiameter    string  `json:"nozzle_diameter"`
	NozzleType        string  `json:"nozzle_type"`

	// Fan speeds (0-15 scale typically)
	CoolingFanSpeed    int `json:"cooling_fan_speed"`
	BigFan1Speed       int `json:"big_fan1_speed"`
	BigFan2Speed       int `json:"big_fan2_speed"`
	HeatbreakFanSpeed  int `json:"heatbreak_fan_speed"`

	// Speeds and flow
	SpeedMag       int     `json:"spd_mag"`
	SpeedLevel     int     `json:"spd_lvl"`
	PrintRealSpeed int     `json:"print_real_speed"`
	FeedRateMag    int     `json:"feed_rate_mag"`

	// AMS (Automatic Material System)
	AMSStatus int            `json:"ams_status"`
	AMS       *BambuAMS      `json:"ams,omitempty"`
	VTTray    *BambuVTTray   `json:"vt_tray,omitempty"`

	// Lights and misc
	LightsReport []BambuLight `json:"lights_report,omitempty"`
	WiFiSignal   string       `json:"wifi_signal"`
	Online       *BambuOnline `json:"online,omitempty"`

	// HMS (Health Management System) errors
	HMS []BambuHMS `json:"hms,omitempty"`

	// Lifecycle
	LifecycleRelayID    string `json:"lifecycle"`
	PrintStartTime      string `json:"gcode_start_time"`
	PrintFinishAction   string `json:"print_finish_action"`
	HomeFlag            int    `json:"home_flag"`
	HWSwitch            int    `json:"hw_switch_state"`
}

// BambuAMS represents AMS (Automatic Material System) status.
type BambuAMS struct {
	AMSExistBits  string      `json:"ams_exist_bits"`
	TrayExistBits string      `json:"tray_exist_bits"`
	TrayNow       string      `json:"tray_now"`
	TrayPre       string      `json:"tray_pre"`
	TrayTar       string      `json:"tray_tar"`
	Version       int         `json:"version"`
	Units         []BambuUnit `json:"ams,omitempty"`
}

// BambuUnit represents a single AMS unit.
type BambuUnit struct {
	ID       string       `json:"id"`
	Humidity string       `json:"humidity"`
	Temp     string       `json:"temp"`
	Trays    []BambuTray  `json:"tray"`
}

// BambuTray represents a tray/slot in an AMS unit.
type BambuTray struct {
	ID            string  `json:"id"`
	TrayIDName    string  `json:"tray_id_name"`
	TrayType      string  `json:"tray_type"`
	TraySubBrands string  `json:"tray_sub_brands"`
	TrayColor     string  `json:"tray_color"`
	TrayWeight    string  `json:"tray_weight"`
	TrayDiameter  string  `json:"tray_diameter"`
	TrayTemp      string  `json:"tray_temp"`
	TrayTime      string  `json:"tray_time"`
	BedTempType   string  `json:"bed_temp_type"`
	BedTemp       string  `json:"bed_temp"`
	NozzleTempMax string  `json:"nozzle_temp_max"`
	NozzleTempMin string  `json:"nozzle_temp_min"`
	Remain        int     `json:"remain"`
	TagUID        string  `json:"tag_uid"`
	TrayUUID      string  `json:"tray_uuid"`
	TrayInfoIdx   string  `json:"tray_info_idx"`
	Cols          []string `json:"cols,omitempty"`
}

// BambuVTTray represents the external spool holder (virtual tray).
type BambuVTTray struct {
	ID            string `json:"id"`
	TrayIDName    string `json:"tray_id_name"`
	TrayType      string `json:"tray_type"`
	TraySubBrands string `json:"tray_sub_brands"`
	TrayColor     string `json:"tray_color"`
	TrayWeight    string `json:"tray_weight"`
	TrayDiameter  string `json:"tray_diameter"`
	TrayTemp      string `json:"tray_temp"`
	TrayTime      string `json:"tray_time"`
	BedTempType   string `json:"bed_temp_type"`
	BedTemp       string `json:"bed_temp"`
	NozzleTempMax string `json:"nozzle_temp_max"`
	NozzleTempMin string `json:"nozzle_temp_min"`
	Remain        int    `json:"remain"`
}

// BambuLight represents a light status report.
type BambuLight struct {
	Node string `json:"node"`
	Mode string `json:"mode"`
}

// BambuOnline represents online status information.
type BambuOnline struct {
	Ahb       bool `json:"ahb"`
	Rfid      bool `json:"rfid"`
	Version   int  `json:"version"`
}

// BambuHMS represents a Health Management System error.
type BambuHMS struct {
	Attr     int    `json:"attr"`
	Code     int    `json:"code"`
	Module   int    `json:"module"`
	Severity int    `json:"severity"`
}

// BambuInfo contains printer information.
type BambuInfo struct {
	Sequence string `json:"sequence_id"`
	Command  string `json:"command"`
	Module   []struct {
		Name    string `json:"name"`
		Project string `json:"project_name"`
		SWVer   string `json:"sw_ver"`
		HWVer   string `json:"hw_ver"`
		SerNo   string `json:"sn"`
	} `json:"module,omitempty"`
}

// BambuSystem contains system information.
type BambuSystem struct {
	Sequence  string `json:"sequence_id"`
	Command   string `json:"command"`
	Result    string `json:"result"`
	Reason    string `json:"reason,omitempty"`
	LedMode   string `json:"led_mode,omitempty"`
	LedOnTime string `json:"led_on_time,omitempty"`
}

// BambuPushing contains push notification info.
type BambuPushing struct {
	Sequence   string `json:"sequence_id"`
	Command    string `json:"command"`
	PushTarget int    `json:"push_target,omitempty"`
	Version    int    `json:"version,omitempty"`
}

// BambuCommand represents a command to send to the printer.
type BambuCommand struct {
	Print   *BambuPrintCmd   `json:"print,omitempty"`
	System  *BambuSystemCmd  `json:"system,omitempty"`
	Pushing *BambuPushingCmd `json:"pushing,omitempty"`
}

// BambuPrintCmd represents a print control command.
type BambuPrintCmd struct {
	Sequence    string       `json:"sequence_id"`
	Command     string       `json:"command"`
	Param       string       `json:"param,omitempty"`
	ProjectID   string       `json:"project_id,omitempty"`
	ProfileID   string       `json:"profile_id,omitempty"`
	TaskID      string       `json:"task_id,omitempty"`
	Subtask     string       `json:"subtask_id,omitempty"`
	SubtaskName string       `json:"subtask_name,omitempty"`
	URL         string       `json:"url,omitempty"`
	Filename    string       `json:"filename,omitempty"`
	Timelapse   bool         `json:"timelapse,omitempty"`
	BedLeveling bool         `json:"bed_leveling,omitempty"`
	FlowCali    bool         `json:"flow_cali,omitempty"`
	Vibration   bool         `json:"vibration_cali,omitempty"`
	AMS         *BambuAMSCmd `json:"ams,omitempty"`
}

// BambuAMSCmd represents AMS settings in a print command.
type BambuAMSCmd struct {
	UseAMS         bool   `json:"use_ams"`
	PrintSpeed     string `json:"print_speed,omitempty"`
	LayerInspect   bool   `json:"layer_inspect,omitempty"`
	LabelVersion   string `json:"label_version,omitempty"`
	AMSMappingInfo []int  `json:"ams_mapping,omitempty"`
}

// BambuSystemCmd represents a system command.
type BambuSystemCmd struct {
	Sequence  string `json:"sequence_id"`
	Command   string `json:"command"`
	LedMode   string `json:"led_mode,omitempty"`
	LedOnTime string `json:"led_on_time,omitempty"`
}

// BambuPushingCmd represents a push command.
type BambuPushingCmd struct {
	Sequence   string `json:"sequence_id"`
	Command    string `json:"command"`
	Version    int    `json:"version,omitempty"`
	PushTarget int    `json:"push_target,omitempty"`
}

// SetLEDMode sets the LED light mode on the printer.
func (c *BambuClient) SetLEDMode(mode string) error {
	cmd := BambuCommand{
		System: &BambuSystemCmd{
			Sequence: strconv.FormatInt(time.Now().UnixMilli(), 10),
			Command:  "ledctrl",
			LedMode:  mode, // "on", "off", "flashing"
		},
	}
	return c.sendCommand(cmd)
}

// SetPrintSpeed sets the print speed profile.
// Levels: 1=silent, 2=standard, 3=sport, 4=ludicrous
func (c *BambuClient) SetPrintSpeed(level int) error {
	cmd := BambuCommand{
		Print: &BambuPrintCmd{
			Sequence: strconv.FormatInt(time.Now().UnixMilli(), 10),
			Command:  "print_speed",
			Param:    strconv.Itoa(level),
		},
	}
	return c.sendCommand(cmd)
}

// GetAMSStatus returns the current AMS filament information.
func (c *BambuClient) GetAMSStatus() (*BambuAMS, error) {
	// This would parse from the last received status
	// For now, request a status update
	c.requestPushAll()
	return nil, fmt.Errorf("not implemented - use status callback")
}
