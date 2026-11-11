package printer

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
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

	"github.com/cadimartech/rankos/internal/model"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
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
	clientID     string // random per session; a stable id gets kicked by another client

	mu         sync.RWMutex
	mqttClient mqtt.Client
	connected  bool
	lastState  *model.PrinterState
	stopChan   chan struct{}

	// pushLoopStarted guards the periodic pushall goroutine.
	pushLoopStarted bool
	// pushAllInterval and reportTimeout override the defaults in tests.
	pushAllInterval time.Duration
	reportTimeout   time.Duration
	// staleAfter and staleInterval override how soon a silent printer is offline.
	staleAfter        time.Duration
	staleInterval     time.Duration
	staleWatchStarted bool
	staleNotified     bool

	modelName         string
	baseCaps          model.Capabilities
	signatureRequired bool
	sawAux            bool
	sawChamberFan     bool
	secondLight       bool
	nozzlesFromReport int
	camera            model.CameraInfo
	cameraSeen        bool
	lastCameraNote    string
	// reportGen increments on each subscribe. reportGenSeen matches it once
	// any report arrives for that generation.
	reportGen      uint64
	reportGenSeen  uint64
	receivedReport bool
}

const (
	// bambuReportTimeout is how long to wait after subscribe before warning
	// that no report arrived. Causes seen in the field are a serial that does
	// not match the printer, or LAN Only / Developer Mode left off on current
	// firmware (H2, P2, and the same control on X, P, and A series).
	bambuReportTimeout = 30 * time.Second
	// bambuPushAllInterval is how often to re-request a full status dump.
	// Every current family sends partial reports, so a periodic pushall
	// refreshes fields the printer does not repeat.
	bambuPushAllInterval = 5 * time.Minute
)

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
		lastState:    initialBambuState(printerID, false),
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
		mqttUsername: mqttUsername,
		stopChan:     make(chan struct{}),
		lastState:    initialBambuState(printerID, true),
	}
}

// SetSerialNumber sets the printer's serial number for MQTT topics.
func (c *BambuClient) SetSerialNumber(serial string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.serialNumber = serial
}

// newBambuClientID returns a fresh MQTT client id.
// A stable id is disconnected when another client (a second RankOS process,
// Bambu Studio, or another farm controller) connects with the same id.
// 23 bytes is the MQTT 3.1 maximum, which some printer firmware still enforces.
func newBambuClientID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("rankos%x", time.Now().UnixNano()%0xffffffff)
	}
	return "rankos-" + hex.EncodeToString(b[:])
}

// bambuMQTTDebug is set by BAMBU_MQTT_DEBUG=1. It subscribes to every printer
// report topic and logs the topic plus the first 300 bytes of each message.
func bambuMQTTDebug() bool {
	return os.Getenv("BAMBU_MQTT_DEBUG") == "1"
}

// mqttOptions builds the client options used for both LAN and cloud.
// Protocol is MQTT 3.1.1, clean session, QoS delivery not ordered. Those match
// ha-bambulab (paho mqtt.Client() defaults) and the OpenBambuAPI LAN notes.
func (c *BambuClient) mqttOptions(broker, clientID, username, password string, tlsConfig *tls.Config) *mqtt.ClientOptions {
	return mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetUsername(username).
		SetPassword(password).
		SetTLSConfig(tlsConfig).
		SetProtocolVersion(4). // MQTT 3.1.1
		SetCleanSession(true).
		SetOrderMatters(false).
		SetAutoReconnect(true).
		SetConnectTimeout(10 * time.Second).
		SetWriteTimeout(10 * time.Second).
		SetKeepAlive(30 * time.Second)
}

// Connect establishes MQTT connection to Bambu printer.
// Connection pattern follows the bambulabs_api community SDK.
// The mutex is not held across the network handshake: the OnConnect handler
// runs on another goroutine and must be able to publish pushall immediately.
func (c *BambuClient) Connect() error {
	c.mu.Lock()
	if c.connected && c.mqttClient != nil && c.mqttClient.IsConnected() {
		c.mu.Unlock()
		return nil
	}
	host := c.host
	accessCode := c.accessCode
	serial := c.serialNumber
	cloudMode := c.cloudMode
	mqttUsername := c.mqttUsername
	printerID := c.printerID
	c.mu.Unlock()

	broker := fmt.Sprintf("ssl://%s:%d", host, BambuMQTTPort)

	// Determine MQTT credentials based on mode
	username := "bblp"
	if cloudMode && mqttUsername != "" {
		username = mqttUsername
	}

	// TLS config differs between local and cloud mode
	var tlsConfig *tls.Config
	if cloudMode {
		// Cloud broker has a valid certificate — use system CA pool
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	} else {
		// Local printer uses a self-signed cert whose CN is the serial, not the IP.
		// ha-bambulab does the same: CERT_NONE and no hostname check.
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		}
	}

	// A fixed client id (printer UUID prefix) collides with another RankOS
	// process or another farm tool and the printer kicks one of them.
	// MQTT 3.1 allows at most 23 bytes; "rankos-" + 16 hex is exactly 23.
	clientID := newBambuClientID()
	c.mu.Lock()
	c.clientID = clientID
	c.mu.Unlock()

	opts := c.mqttOptions(broker, clientID, username, accessCode, tlsConfig)

	// Use DefaultPublishHandler for all incoming messages (matches community SDK pattern)
	opts.SetDefaultPublishHandler(c.handleMessage)
	opts.SetOnConnectHandler(c.onConnect)
	opts.SetConnectionLostHandler(c.onConnectionLost)
	opts.SetReconnectingHandler(c.onReconnecting)

	client := mqtt.NewClient(opts)

	mode := "LAN"
	if cloudMode {
		mode = "cloud"
	}
	slog.Info("connecting to Bambu printer",
		"mode", mode,
		"host", host,
		"printer_id", printerID,
		"serial", serial,
		"broker", broker,
		"username", username,
		"client_id", clientID,
		"mqtt_protocol", "3.1.1",
		"clean_session", true,
		"order_matters", false,
	)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("MQTT connection failed: %w", token.Error())
	}

	c.mu.Lock()
	c.mqttClient = client
	c.connected = true
	c.mu.Unlock()
	slog.Info("connected to Bambu printer", "mode", mode, "host", host, "printer_id", printerID, "client_id", clientID, "serial", serial)

	return nil
}

// onConnect is called when MQTT connection is established (including paho
// auto-reconnect). It subscribes, immediately requests a full status dump,
// and arms the silence warning. Current Bambu firmware does not volunteer a
// full snapshot; without pushall the connection stays up and status stays offline.
func (c *BambuClient) onConnect(client mqtt.Client) {
	c.mu.Lock()
	c.mqttClient = client
	c.connected = true
	c.receivedReport = false
	c.reportGen++
	serial := c.serialNumber
	host := c.host
	accessLen := len(c.accessCode)
	hasAccess := c.accessCode != ""
	c.mu.Unlock()

	slog.Info("Bambu MQTT connected",
		"printer_id", c.printerID,
		"host", host,
		"serial", serial,
		"client_id", c.clientID,
		"has_access_code", hasAccess,
		"access_code_len", accessLen,
	)

	// Pass the handler to Subscribe. A nil callback relies on
	// DefaultPublishHandler; paho will not call that default for a topic that
	// has its own route, so the handler has to be attached here.
	topic := fmt.Sprintf("device/%s/report", serial)
	if err := c.subscribe(client, topic, c.handleMessage); err != nil {
		slog.Error("failed to subscribe to Bambu report topic",
			"error", err,
			"topic", topic,
			"serial", serial,
			"client_id", c.clientID,
		)
	}

	if bambuMQTTDebug() {
		// Diagnostic: the printer may be publishing on a different serial than
		// the one we stored. The wildcard shows the real topic.
		if err := c.subscribe(client, "device/+/report", c.debugReport); err != nil {
			slog.Error("failed to subscribe to Bambu debug wildcard",
				"error", err,
				"topic", "device/+/report",
				"client_id", c.clientID,
			)
		}
	}

	// Arm before pushall so a fast report is attributed to this generation.
	c.armNoReportWatch()

	// ha-bambulab asks for the version, then a full dump. The dump must be
	// exactly {"pushing":{"sequence_id":"0","command":"pushall"}}. That client
	// uses this payload for every model it supports, including P2S and H2D.
	c.publishInitial(client, serial)
	c.ensurePushLoop()
	c.ensureStaleWatch()
}

// subscribe sends SUBSCRIBE and logs the SUBACK return code.
// 0x80 means the printer refused the subscription; token.Error() stays nil
// in that case, so the code has to be read off the token result.
func (c *BambuClient) subscribe(client mqtt.Client, topic string, handler mqtt.MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("refusing to subscribe to %s without a message handler", topic)
	}
	token := client.Subscribe(topic, 0, handler)
	if token == nil {
		return fmt.Errorf("subscribe returned a nil token")
	}
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("subscribe timeout")
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("subscribe token error: %w", err)
	}
	codes := subscribeReturnCodes(token)
	if len(codes) == 0 {
		slog.Warn("Bambu SUBACK missing return code", "topic", topic, "printer_id", c.printerID, "client_id", c.clientID)
		return nil
	}
	for t, code := range codes {
		if code == 0x80 {
			slog.Error("Bambu subscription refused",
				"topic", t,
				"return_code", "0x80",
				"printer_id", c.printerID,
				"serial", c.serialNumber,
				"client_id", c.clientID,
			)
			return fmt.Errorf("subscription refused for %s (SUBACK 0x80)", t)
		}
		slog.Info("Bambu SUBACK",
			"topic", t,
			"return_code", fmt.Sprintf("0x%02x", code),
			"granted_qos", code,
			"printer_id", c.printerID,
			"client_id", c.clientID,
		)
	}
	return nil
}

// subscribeReturnCodes reads SUBACK codes when the token exposes them.
// paho's *SubscribeToken does; a refused subscription is 0x80 per topic.
func subscribeReturnCodes(token mqtt.Token) map[string]byte {
	type resultToken interface {
		Result() map[string]byte
	}
	rt, ok := token.(resultToken)
	if !ok || rt.Result() == nil {
		return nil
	}
	return rt.Result()
}

// onConnectionLost is called when MQTT connection is lost.
func (c *BambuClient) onConnectionLost(client mqtt.Client, err error) {
	reason := "unknown"
	if err != nil {
		reason = err.Error()
	}
	slog.Warn("Bambu MQTT connection lost",
		"printer_id", c.printerID,
		"client_id", c.clientID,
		"serial", c.serialNumber,
		"host", c.host,
		"reason", reason,
		"error", err,
	)

	c.mu.Lock()
	c.connected = false
	if c.lastState != nil {
		c.lastState.Status = model.PrinterStatusOffline
		c.lastState.UpdatedAt = time.Now()
	}
	state := c.snapshotLocked(time.Now())
	cb := c.statusCallback
	c.mu.Unlock()

	if cb != nil {
		cb(state)
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
	if bambuMQTTDebug() {
		logBambuDebugMessage(msg.Topic(), msg.Payload())
	}
	c.ingestReport(msg.Topic(), msg.Payload())
}

// debugReport logs wildcard traffic. Messages for our own serial are also
// delivered to handleMessage via the specific subscription, so this does not
// ingest them again.
func (c *BambuClient) debugReport(_ mqtt.Client, msg mqtt.Message) {
	logBambuDebugMessage(msg.Topic(), msg.Payload())
}

// logBambuDebugMessage logs the topic and the first 300 bytes of a report.
func logBambuDebugMessage(topic string, payload []byte) {
	n := len(payload)
	shown := payload
	if n > 300 {
		shown = payload[:300]
	}
	slog.Info("Bambu MQTT debug message", "topic", topic, "bytes", n, "payload", string(shown))
}

// ingestReport records that the printer spoke, merges a print payload into
// state, and notifies listeners. Any valid report clears the post-subscribe
// silence warning.
func (c *BambuClient) ingestReport(topic string, payload []byte) {
	var msg BambuMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		slog.Debug("failed to parse Bambu message", "error", err, "payload", string(payload))
		return
	}

	c.mu.Lock()
	first := !c.receivedReport
	c.receivedReport = true
	c.staleNotified = false
	c.reportGenSeen = c.reportGen
	var state *model.PrinterState
	if msg.Print != nil {
		state = c.mergePrintStatus(msg.Print)
		c.lastState = state
	}
	cb := c.statusCallback
	c.mu.Unlock()

	if first {
		slog.Debug("first Bambu report received",
			"printer_id", c.printerID,
			"topic", topic,
			"bytes", len(payload),
		)
	}

	if state != nil && cb != nil {
		cb(state)
	}
}

// pushAllCommand matches ha-bambulab PUSH_ALL exactly. version and push_target
// are not in that payload; the same request is what current LAN clients send
// for P1, P2, A1, H2, and X1.
func pushAllCommand() BambuCommand {
	return pushingCommand("pushall")
}

// versionCommand matches ha-bambulab GET_VERSION, sent just before pushall.
func versionCommand() BambuCommand {
	return BambuCommand{
		Info: &BambuInfoCmd{
			Sequence: "0",
			Command:  "get_version",
		},
	}
}

// pushingCommand builds a pushing request such as pushall or start.
func pushingCommand(command string) BambuCommand {
	return BambuCommand{
		Pushing: &BambuPushingCmd{
			Sequence: "0",
			Command:  command,
		},
	}
}

// publishInitial asks for the firmware version and a full status dump.
func (c *BambuClient) publishInitial(client mqtt.Client, serial string) {
	if err := c.publishCommand(client, serial, versionCommand()); err != nil {
		slog.Error("failed to publish Bambu get_version",
			"error", err,
			"token_error", err.Error(),
			"printer_id", c.printerID,
			"serial", serial,
			"client_id", c.clientID,
			"topic", fmt.Sprintf("device/%s/request", serial),
		)
	}
	if err := c.publishPushAll(client, serial); err != nil {
		slog.Error("failed to publish Bambu pushall",
			"error", err,
			"token_error", err.Error(),
			"printer_id", c.printerID,
			"serial", serial,
			"client_id", c.clientID,
			"topic", fmt.Sprintf("device/%s/request", serial),
		)
	}
}

// publishPushAll sends pushall to device/<serial>/request.
// QoS 0 because the printer does not PUBACK.
func (c *BambuClient) publishPushAll(client mqtt.Client, serial string) error {
	if err := c.publishCommand(client, serial, pushAllCommand()); err != nil {
		return err
	}
	return nil
}

// publishCommand publishes a request and surfaces the token error.
func (c *BambuClient) publishCommand(client mqtt.Client, serial string, cmd BambuCommand) error {
	if client == nil {
		return fmt.Errorf("not connected")
	}
	if serial == "" {
		serial = "unknown"
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}
	topic := fmt.Sprintf("device/%s/request", serial)
	token := client.Publish(topic, 0, false, data)
	if token == nil {
		return fmt.Errorf("publish token error: nil token")
	}
	if !token.WaitTimeout(5 * time.Second) {
		if err := token.Error(); err != nil {
			return fmt.Errorf("publish token error: %w", err)
		}
		return fmt.Errorf("publish token error: timeout on %s", topic)
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("publish token error: %w", err)
	}
	slog.Info("published Bambu command",
		"topic", topic,
		"payload", string(data),
		"printer_id", c.printerID,
		"serial", serial,
		"client_id", c.clientID,
	)
	if cmd.Pushing != nil && cmd.Pushing.Command == "pushall" {
		slog.Info("published Bambu pushall", "topic", topic, "printer_id", c.printerID, "serial", serial, "client_id", c.clientID)
	}
	return nil
}

// requestPushAll requests the printer to send all current status.
func (c *BambuClient) requestPushAll() {
	c.mu.RLock()
	client := c.mqttClient
	serial := c.serialNumber
	connected := c.connected && client != nil
	c.mu.RUnlock()
	if !connected {
		slog.Warn("skipped Bambu pushall; client not connected", "printer_id", c.printerID, "serial", serial)
		return
	}
	if err := c.publishPushAll(client, serial); err != nil {
		slog.Warn("Bambu pushall failed", "error", err, "printer_id", c.printerID, "serial", serial)
	}
}

// ensurePushLoop republishes pushall on an interval. onConnect publishes
// immediately, including after paho auto-reconnect; this covers the gap.
func (c *BambuClient) ensurePushLoop() {
	c.mu.Lock()
	if c.pushLoopStarted {
		c.mu.Unlock()
		return
	}
	c.pushLoopStarted = true
	interval := c.pushAllInterval
	if interval <= 0 {
		interval = bambuPushAllInterval
	}
	stop := c.stopChan
	c.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				c.mu.RLock()
				client := c.mqttClient
				serial := c.serialNumber
				connected := c.connected && client != nil
				c.mu.RUnlock()
				if !connected {
					continue
				}
				if err := c.publishPushAll(client, serial); err != nil {
					slog.Warn("periodic Bambu pushall failed", "error", err, "printer_id", c.printerID, "serial", serial)
				}
			}
		}
	}()
}

// armNoReportWatch logs a warning when the printer accepts the subscription
// but never sends a report. That is a wrong serial, or LAN Only / Developer
// Mode left off so current firmware does not open the MQTT report channel.
func (c *BambuClient) armNoReportWatch() {
	c.mu.Lock()
	// onConnect advances reportGen before subscribing. Tests call this
	// directly and start from zero.
	if c.reportGen == 0 {
		c.reportGen = 1
	}
	gen := c.reportGen
	timeout := c.reportTimeout
	if timeout <= 0 {
		timeout = bambuReportTimeout
	}
	serial := c.serialNumber
	stop := c.stopChan
	c.mu.Unlock()

	go func() {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-stop:
			return
		case <-timer.C:
		}
		c.mu.RLock()
		current := c.reportGen
		got := c.reportGenSeen == gen
		c.mu.RUnlock()
		if current != gen || got {
			return
		}
		slog.Warn("no reports received; check that the serial matches the printer and that LAN Only and Developer Mode are enabled",
			"printer_id", c.printerID,
			"serial", serial,
			"client_id", c.clientID,
			"topic", fmt.Sprintf("device/%s/report", serial),
		)
		// ha-bambulab's watchdog sends pushing.command "start" when any model
		// stays silent. Some firmware waits for that before streaming.
		c.mu.RLock()
		client := c.mqttClient
		curSerial := c.serialNumber
		connected := c.connected && client != nil
		c.mu.RUnlock()
		if !connected {
			return
		}
		if err := c.publishCommand(client, curSerial, pushingCommand("start")); err != nil {
			slog.Error("failed to publish Bambu start",
				"error", err,
				"token_error", err.Error(),
				"printer_id", c.printerID,
				"serial", curSerial,
				"client_id", c.clientID,
			)
		}
	}()
}

// sendCommand sends a command to the printer via MQTT.
func (c *BambuClient) sendCommand(cmd BambuCommand) error {
	c.mu.RLock()
	if !c.connected || c.mqttClient == nil {
		c.mu.RUnlock()
		return ErrNotConnected
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
// A report older than ReportStaleAfter is returned as offline without forgetting
// the last temperatures, so the next delta can still merge.
func (c *BambuClient) GetStatus() (*model.PrinterState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshotLocked(time.Now()), nil
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
				UseAMS:         true,
				PrintSpeed:     "standard",
				LayerInspect:   false,
				LabelVersion:   "",
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
// The command is not tied to a RankOS job. It pauses whatever the printer is running.
func (c *BambuClient) PauseJob() error {
	return c.Send(model.Pause{})
}

// ResumeJob resumes a paused print via MQTT command.
func (c *BambuClient) ResumeJob() error {
	return c.Send(model.Resume{})
}

// CancelJob stops the current print via MQTT command.
func (c *BambuClient) CancelJob() error {
	return c.Send(model.Stop{})
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
	state := c.cloneState()
	if print != nil && print.present != nil {
		c.applyPresentPrintStatus(state, print)
	} else if print != nil {
		c.applyLegacyPrintStatus(state, print)
	}
	// Any report means the printer is on the wire. Deltas on every current
	// family often omit gcode_state, which used to leave status stuck at offline.
	if state.Status == model.PrinterStatusOffline {
		state.Status = model.PrinterStatusIdle
	}
	c.absorb(state, print)
	model.RefreshView(state, time.Now())
	return state
}

// cloneState copies the last known state so a partial report can fill in
// only the fields it actually carries.
func (c *BambuClient) cloneState() *model.PrinterState {
	if c.lastState == nil {
		return initialBambuState(c.printerID, c.cloudMode)
	}
	state := *c.lastState
	state.PrinterID = c.printerID
	state.UpdatedAt = time.Now()
	return &state
}

// applyLegacyPrintStatus updates state using non-zero heuristics. Used when
// a status struct was built in code (no JSON presence map).
func (c *BambuClient) applyLegacyPrintStatus(state *model.PrinterState, print *BambuPrintStatus) {
	// Only update status when gcode_state is actually present
	if print.GcodeState != "" {
		state.GcodeState = print.GcodeState
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
	if print.PrintError != 0 || print.GcodeState != "" {
		state.PrintError = print.PrintError
	}
	if print.StgCur != 0 {
		state.Stage = stageLabel(print.StgCur)
	}

	// Parse AMS state
	if print.AMS != nil {
		state.AMS = c.parseAMSState(print.AMS, print.VTTray)
	}
}

// applyPresentPrintStatus merges a JSON report using only keys that were
// actually present. Bambu printers send deltas; missing keys must not wipe
// the last known temperatures, progress, or file name, and explicit zeros
// (heater off, fan off, progress reset) must still apply. The same merge is
// used for every family. H2 dual-nozzle detail beyond nozzle_temper is not
// parsed; unknown keys are ignored and do not clear the fields we do track.
func (c *BambuClient) applyPresentPrintStatus(state *model.PrinterState, print *BambuPrintStatus) {
	if print.has("gcode_state") && print.GcodeState != "" {
		state.GcodeState = print.GcodeState
		switch print.GcodeState {
		case "IDLE", "FINISH":
			state.Status = model.PrinterStatusIdle
		case "RUNNING", "PREPARE":
			state.Status = model.PrinterStatusPrinting
		case "PAUSE":
			state.Status = model.PrinterStatusPaused
		case "FAILED":
			state.Status = model.PrinterStatusError
		}
	} else if print.has("stg_cur") && print.StgCur > 0 && print.StgCur < 255 {
		state.Status = model.PrinterStatusPrinting
	}

	if print.has("mc_percent") {
		state.Progress = float64(print.MCPercent)
	}
	if print.has("subtask_name") && print.SubtaskName != "" {
		state.CurrentFile = print.SubtaskName
	} else if print.has("gcode_file") && print.GcodeFile != "" {
		state.CurrentFile = print.GcodeFile
	}
	if print.has("mc_remaining_time") {
		state.TimeLeft = print.MCRemainingTime * 60
	}
	if print.has("bed_temper") {
		state.BedTemp = print.BedTemper
	}
	if print.has("nozzle_temper") {
		state.NozzleTemp = print.NozzleTemper
	}
	if print.has("bed_target_temper") {
		state.BedTargetTemp = print.BedTargetTemper
	}
	if print.has("nozzle_target_temper") {
		state.NozzleTargetTemp = print.NozzleTargetTemper
	}
	if print.has("chamber_temper") {
		state.ChamberTemp = print.ChamberTemper
	}
	if print.has("layer_num") {
		state.LayerNum = print.LayerNum
	}
	if print.has("total_layer_num") {
		state.TotalLayerNum = print.TotalLayerNum
	}
	if print.has("cooling_fan_speed") {
		state.CoolingFanSpeed = bambuFanToPercent(print.CoolingFanSpeed)
	}
	if print.has("big_fan1_speed") {
		state.AuxFanSpeed = bambuFanToPercent(print.BigFan1Speed)
	}
	if print.has("big_fan2_speed") {
		state.ChamberFanSpeed = bambuFanToPercent(print.BigFan2Speed)
	}
	if print.has("heatbreak_fan_speed") {
		state.HeatbreakFanSpeed = bambuFanToPercent(print.HeatbreakFanSpeed)
	}
	if print.has("spd_mag") {
		state.SpeedPercent = print.SpeedMag
	}
	if print.has("spd_lvl") {
		state.SpeedLevel = print.SpeedLevel
	}
	if print.has("print_real_speed") {
		state.PrintRealSpeed = print.PrintRealSpeed
	}
	if print.has("wifi_signal") {
		state.WiFiSignal = print.WiFiSignal
	}
	if print.has("nozzle_diameter") {
		state.NozzleDiameter = print.NozzleDiameter
	}
	if print.has("nozzle_type") {
		state.NozzleType = print.NozzleType
	}
	if print.has("hms") {
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
	if print.has("lights_report") {
		state.Lights = make([]model.LightState, len(print.LightsReport))
		for i, l := range print.LightsReport {
			state.Lights[i] = model.LightState{
				Node: l.Node,
				Mode: l.Mode,
			}
		}
	}
	if print.has("gcode_start_time") {
		state.GcodeStartTime = print.PrintStartTime
	}
	if print.has("subtask_id") {
		state.SubtaskID = print.SubtaskID
	}
	if print.has("task_id") {
		state.TaskID = print.TaskID
	}
	if print.has("print_type") {
		state.PrintType = print.PrintType
	}
	if print.has("print_error") {
		state.PrintError = print.PrintError
	}
	if print.has("stg_cur") {
		state.Stage = stageLabel(print.StgCur)
	}
	if print.has("ams") && print.AMS != nil {
		state.AMS = c.parseAMSState(print.AMS, print.VTTray)
	}
	if print.has("gcode_state") && (print.GcodeState == "IDLE" || print.GcodeState == "FINISH") {
		state.LayerNum = 0
		state.TotalLayerNum = 0
		state.TimeLeft = 0
		state.Progress = 0
		if print.has("mc_percent") {
			state.Progress = float64(print.MCPercent)
		}
	}
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
	// present records JSON keys from the wire so partial deltas do not look
	// like explicit zeros. Nil when the struct was built in code.
	present map[string]json.RawMessage `json:"-"`

	// Print state
	GcodeState  string `json:"gcode_state"`
	StgCur      int    `json:"stg_cur"`
	PrintError  int    `json:"print_error"`
	PrintType   string `json:"print_type"`
	SubtaskName string `json:"subtask_name"`
	GcodeFile   string `json:"gcode_file"`
	SubtaskID   string `json:"subtask_id"`
	TaskID      string `json:"task_id"`

	// Progress
	MCPercent       int `json:"mc_percent"`
	MCRemainingTime int `json:"mc_remaining_time"`
	LayerNum        int `json:"layer_num"`
	TotalLayerNum   int `json:"total_layer_num"`

	// Temperatures
	BedTemper          float64 `json:"bed_temper"`
	BedTargetTemper    float64 `json:"bed_target_temper"`
	NozzleTemper       float64 `json:"nozzle_temper"`
	NozzleTargetTemper float64 `json:"nozzle_target_temper"`
	ChamberTemper      float64 `json:"chamber_temper"`
	NozzleDiameter     string  `json:"nozzle_diameter"`
	NozzleType         string  `json:"nozzle_type"`

	// Fan speeds (0-15 scale typically)
	CoolingFanSpeed   int `json:"cooling_fan_speed"`
	BigFan1Speed      int `json:"big_fan1_speed"`
	BigFan2Speed      int `json:"big_fan2_speed"`
	HeatbreakFanSpeed int `json:"heatbreak_fan_speed"`

	// Speeds and flow
	SpeedMag       int `json:"spd_mag"`
	SpeedLevel     int `json:"spd_lvl"`
	PrintRealSpeed int `json:"print_real_speed"`
	FeedRateMag    int `json:"feed_rate_mag"`

	// AMS (Automatic Material System)
	AMSStatus int          `json:"ams_status"`
	AMS       *BambuAMS    `json:"ams,omitempty"`
	VTTray    *BambuVTTray `json:"vt_tray,omitempty"`

	// Lights and misc
	LightsReport []BambuLight `json:"lights_report,omitempty"`
	WiFiSignal   string       `json:"wifi_signal"`
	Online       *BambuOnline `json:"online,omitempty"`

	// HMS (Health Management System) errors
	HMS []BambuHMS `json:"hms,omitempty"`

	// Lifecycle
	LifecycleRelayID  string `json:"lifecycle"`
	PrintStartTime    string `json:"gcode_start_time"`
	PrintFinishAction string `json:"print_finish_action"`
	HomeFlag          int    `json:"home_flag"`
	HWSwitch          int    `json:"hw_switch_state"`

	// Security and camera. fun is a hex bitfield. flag3 is the fallback when
	// fun is empty. ipcam decides camera availability.
	Fun    string          `json:"fun"`
	Flag3  int             `json:"flag3"`
	IPCam  *BambuIPCam     `json:"ipcam,omitempty"`
	Device json.RawMessage `json:"device,omitempty"`
}

// BambuIPCam is the camera block inside a print report.
type BambuIPCam struct {
	RTSPURL  string `json:"rtsp_url"`
	Liveview *struct {
		Local string `json:"local"`
	} `json:"liveview"`
}

// UnmarshalJSON captures which keys the printer actually sent.
func (p *BambuPrintStatus) UnmarshalJSON(data []byte) error {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}
	// A null fun or flag3 must not reject the rest of the report.
	if raw, ok := keys["fun"]; ok && strings.TrimSpace(string(raw)) == "null" {
		keys["fun"] = []byte(`""`)
	}
	if raw, ok := keys["flag3"]; ok && strings.TrimSpace(string(raw)) == "null" {
		delete(keys, "flag3")
	}
	// Some firmware sends numbers as strings (observed on A1-class reports).
	// Coerce those so one odd field does not reject the whole report.
	encoded, err := json.Marshal(relaxNumericStrings(keys))
	if err != nil {
		return err
	}
	type plain BambuPrintStatus
	var decoded plain
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return err
	}
	*p = BambuPrintStatus(decoded)
	p.present = keys
	return nil
}

// relaxNumericStrings copies keys, turning JSON strings that are plain numbers
// into JSON numbers. Nested objects are left alone.
func relaxNumericStrings(keys map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(keys))
	for k, raw := range keys {
		out[k] = raw
		if len(raw) < 2 || raw[0] != '"' {
			continue
		}
		s, err := strconv.Unquote(string(raw))
		if err != nil || s == "" {
			continue
		}
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			continue
		}
		out[k] = json.RawMessage(s)
	}
	return out
}

// has reports whether key was present in the JSON payload.
func (p *BambuPrintStatus) has(key string) bool {
	if p == nil || p.present == nil {
		return false
	}
	_, ok := p.present[key]
	return ok
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
	ID       string      `json:"id"`
	Humidity string      `json:"humidity"`
	Temp     string      `json:"temp"`
	Trays    []BambuTray `json:"tray"`
}

// BambuTray represents a tray/slot in an AMS unit.
type BambuTray struct {
	ID            string   `json:"id"`
	TrayIDName    string   `json:"tray_id_name"`
	TrayType      string   `json:"tray_type"`
	TraySubBrands string   `json:"tray_sub_brands"`
	TrayColor     string   `json:"tray_color"`
	TrayWeight    string   `json:"tray_weight"`
	TrayDiameter  string   `json:"tray_diameter"`
	TrayTemp      string   `json:"tray_temp"`
	TrayTime      string   `json:"tray_time"`
	BedTempType   string   `json:"bed_temp_type"`
	BedTemp       string   `json:"bed_temp"`
	NozzleTempMax string   `json:"nozzle_temp_max"`
	NozzleTempMin string   `json:"nozzle_temp_min"`
	Remain        int      `json:"remain"`
	TagUID        string   `json:"tag_uid"`
	TrayUUID      string   `json:"tray_uuid"`
	TrayInfoIdx   string   `json:"tray_info_idx"`
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
	Ahb     bool `json:"ahb"`
	Rfid    bool `json:"rfid"`
	Version int  `json:"version"`
}

// BambuHMS represents a Health Management System error.
type BambuHMS struct {
	Attr     int `json:"attr"`
	Code     int `json:"code"`
	Module   int `json:"module"`
	Severity int `json:"severity"`
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
	Info    *BambuInfoCmd    `json:"info,omitempty"`
}

// BambuInfoCmd is an outgoing info request such as get_version.
type BambuInfoCmd struct {
	Sequence string `json:"sequence_id"`
	Command  string `json:"command"`
}

// BambuPrintCmd represents a print control command.
type BambuPrintCmd struct {
	Sequence    string       `json:"sequence_id"`
	Command     string       `json:"command"`
	Param       *string      `json:"param,omitempty"`
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
// The blink fields are required on every ledctrl, including a steady on or off.
type BambuSystemCmd struct {
	Sequence     string `json:"sequence_id"`
	Command      string `json:"command"`
	LedNode      string `json:"led_node,omitempty"`
	LedMode      string `json:"led_mode,omitempty"`
	LedOnTime    int    `json:"led_on_time,omitempty"`
	LedOffTime   int    `json:"led_off_time,omitempty"`
	LoopTimes    int    `json:"loop_times,omitempty"`
	IntervalTime int    `json:"interval_time,omitempty"`
}

// BambuPushingCmd represents a push command.
type BambuPushingCmd struct {
	Sequence   string `json:"sequence_id"`
	Command    string `json:"command"`
	Version    int    `json:"version,omitempty"`
	PushTarget int    `json:"push_target,omitempty"`
}

// SetLEDMode turns the chamber light on or off.
// mode "off" switches it off. Any other value switches it on.
// The MQTT payload includes led_node and the blink fields OpenBambuAPI requires.
func (c *BambuClient) SetLEDMode(mode string) error {
	return c.Send(model.SetLight{Node: model.LightChamber, On: mode != "off"})
}

// SetPrintSpeed sets the print speed profile.
// Levels: 1=silent, 2=standard, 3=sport, 4=ludicrous
func (c *BambuClient) SetPrintSpeed(level int) error {
	return c.Send(model.SetSpeed{Level: level})
}

// GetAMSStatus returns the current AMS filament information.
func (c *BambuClient) GetAMSStatus() (*BambuAMS, error) {
	// This would parse from the last received status
	// For now, request a status update
	c.requestPushAll()
	return nil, fmt.Errorf("not implemented - use status callback")
}
