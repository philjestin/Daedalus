package printer

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cadimartech/rankos/internal/model"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

func strPtr(s string) *string { return &s }

func TestNewBambuClient(t *testing.T) {
	printerID := uuid.New()
	host := "192.168.1.100"
	accessCode := "12345678"

	client := NewBambuClient(printerID, host, accessCode, "")

	if client.printerID != printerID {
		t.Errorf("expected printer ID %s, got %s", printerID, client.printerID)
	}
	if client.host != host {
		t.Errorf("expected host %s, got %s", host, client.host)
	}
	if client.accessCode != accessCode {
		t.Errorf("expected access code %s, got %s", accessCode, client.accessCode)
	}
	if client.serialNumber != "unknown" {
		t.Errorf("expected serial number 'unknown', got %s", client.serialNumber)
	}
}

func TestNewBambuClient_WithSerialNumber(t *testing.T) {
	printerID := uuid.New()
	host := "192.168.1.100"
	accessCode := "12345678"

	client := NewBambuClient(printerID, host, accessCode, "00M09A350100123")

	if client.serialNumber != "00M09A350100123" {
		t.Errorf("expected serial number '00M09A350100123', got %s", client.serialNumber)
	}
}

func TestSetSerialNumber(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.100", "12345678", "")

	client.SetSerialNumber("00M09A350100456")

	if client.serialNumber != "00M09A350100456" {
		t.Errorf("expected serial number '00M09A350100456', got %s", client.serialNumber)
	}
}

func TestParsePrintStatus_Idle(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.100", "12345678", "")

	printStatus := &BambuPrintStatus{
		GcodeState:   "IDLE",
		BedTemper:    25.0,
		NozzleTemper: 28.0,
	}

	state := client.mergePrintStatus(printStatus)

	if state.Status != model.PrinterStatusIdle {
		t.Errorf("expected status Idle, got %s", state.Status)
	}
	if state.BedTemp != 25.0 {
		t.Errorf("expected bed temp 25.0, got %f", state.BedTemp)
	}
	if state.NozzleTemp != 28.0 {
		t.Errorf("expected nozzle temp 28.0, got %f", state.NozzleTemp)
	}
}

func TestParsePrintStatus_Printing(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.100", "12345678", "")

	printStatus := &BambuPrintStatus{
		GcodeState:         "RUNNING",
		MCPercent:          45,
		MCRemainingTime:    120, // 120 minutes
		SubtaskName:        "test_print.3mf",
		BedTemper:          60.0,
		BedTargetTemper:    60.0,
		NozzleTemper:       210.0,
		NozzleTargetTemper: 215.0,
	}

	state := client.mergePrintStatus(printStatus)

	if state.Status != model.PrinterStatusPrinting {
		t.Errorf("expected status Printing, got %s", state.Status)
	}
	if state.Progress != 45.0 {
		t.Errorf("expected progress 45.0, got %f", state.Progress)
	}
	if state.TimeLeft != 7200 { // 120 minutes * 60 seconds
		t.Errorf("expected time left 7200, got %d", state.TimeLeft)
	}
	if state.CurrentFile != "test_print.3mf" {
		t.Errorf("expected current file 'test_print.3mf', got %s", state.CurrentFile)
	}
}

func TestParsePrintStatus_Paused(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.100", "12345678", "")

	printStatus := &BambuPrintStatus{
		GcodeState:  "PAUSE",
		MCPercent:   50,
		SubtaskName: "paused_print.3mf",
	}

	state := client.mergePrintStatus(printStatus)

	if state.Status != model.PrinterStatusPaused {
		t.Errorf("expected status Paused, got %s", state.Status)
	}
}

func TestParsePrintStatus_Failed(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.100", "12345678", "")

	printStatus := &BambuPrintStatus{
		GcodeState: "FAILED",
		PrintError: 1,
	}

	state := client.mergePrintStatus(printStatus)

	if state.Status != model.PrinterStatusError {
		t.Errorf("expected status Error, got %s", state.Status)
	}
}

func TestParsePrintStatus_Prepare(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.100", "12345678", "")

	printStatus := &BambuPrintStatus{
		GcodeState:  "PREPARE",
		SubtaskName: "preparing_print.3mf",
	}

	state := client.mergePrintStatus(printStatus)

	if state.Status != model.PrinterStatusPrinting {
		t.Errorf("expected status Printing (preparing), got %s", state.Status)
	}
}

func TestParsePrintStatus_Finish(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.100", "12345678", "")

	printStatus := &BambuPrintStatus{
		GcodeState: "FINISH",
		MCPercent:  100,
	}

	state := client.mergePrintStatus(printStatus)

	if state.Status != model.PrinterStatusIdle {
		t.Errorf("expected status Idle (finished), got %s", state.Status)
	}
}

func TestParsePrintStatus_UnknownStateWithStgCur(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.100", "12345678", "")

	// When GcodeState is unknown but stg_cur indicates activity
	printStatus := &BambuPrintStatus{
		GcodeState: "",
		StgCur:     5, // Some intermediate stage
	}

	state := client.mergePrintStatus(printStatus)

	if state.Status != model.PrinterStatusPrinting {
		t.Errorf("expected status Printing (due to stg_cur), got %s", state.Status)
	}
}

func TestParsePrintStatus_GcodeFileFallback(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.100", "12345678", "")

	printStatus := &BambuPrintStatus{
		GcodeState: "RUNNING",
		GcodeFile:  "fallback_file.gcode",
	}

	state := client.mergePrintStatus(printStatus)

	if state.CurrentFile != "fallback_file.gcode" {
		t.Errorf("expected current file 'fallback_file.gcode', got %s", state.CurrentFile)
	}
}

func TestBambuMessageParsing(t *testing.T) {
	jsonData := `{
		"print": {
			"gcode_state": "RUNNING",
			"mc_percent": 75,
			"mc_remaining_time": 30,
			"subtask_name": "benchy.3mf",
			"bed_temper": 60.5,
			"bed_target_temper": 60.0,
			"nozzle_temper": 215.2,
			"nozzle_target_temper": 220.0,
			"layer_num": 150,
			"total_layer_num": 200,
			"cooling_fan_speed": 15,
			"spd_lvl": 2
		}
	}`

	var msg BambuMessage
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if msg.Print == nil {
		t.Fatal("expected print status to be present")
	}
	if msg.Print.GcodeState != "RUNNING" {
		t.Errorf("expected gcode_state 'RUNNING', got %s", msg.Print.GcodeState)
	}
	if msg.Print.MCPercent != 75 {
		t.Errorf("expected mc_percent 75, got %d", msg.Print.MCPercent)
	}
	if msg.Print.LayerNum != 150 {
		t.Errorf("expected layer_num 150, got %d", msg.Print.LayerNum)
	}
	if msg.Print.TotalLayerNum != 200 {
		t.Errorf("expected total_layer_num 200, got %d", msg.Print.TotalLayerNum)
	}
}

func TestBambuMessageParsing_WithAMS(t *testing.T) {
	jsonData := `{
		"print": {
			"gcode_state": "IDLE",
			"ams_status": 0,
			"ams": {
				"ams_exist_bits": "1",
				"tray_exist_bits": "15",
				"tray_now": "0",
				"version": 1,
				"ams": [
					{
						"id": "0",
						"humidity": "3",
						"temp": "25.0",
						"tray": [
							{
								"id": "0",
								"tray_type": "PLA",
								"tray_color": "FFFFFFFF",
								"remain": 85
							}
						]
					}
				]
			}
		}
	}`

	var msg BambuMessage
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if msg.Print.AMS == nil {
		t.Fatal("expected AMS to be present")
	}
	if msg.Print.AMS.AMSExistBits != "1" {
		t.Errorf("expected ams_exist_bits '1', got %s", msg.Print.AMS.AMSExistBits)
	}
	if len(msg.Print.AMS.Units) != 1 {
		t.Errorf("expected 1 AMS unit, got %d", len(msg.Print.AMS.Units))
	}
	if len(msg.Print.AMS.Units[0].Trays) != 1 {
		t.Errorf("expected 1 tray, got %d", len(msg.Print.AMS.Units[0].Trays))
	}
	if msg.Print.AMS.Units[0].Trays[0].TrayType != "PLA" {
		t.Errorf("expected tray type 'PLA', got %s", msg.Print.AMS.Units[0].Trays[0].TrayType)
	}
}

func TestBambuMessageParsing_WithHMS(t *testing.T) {
	jsonData := `{
		"print": {
			"gcode_state": "FAILED",
			"hms": [
				{
					"attr": 1,
					"code": 100,
					"module": 2,
					"severity": 3
				}
			]
		}
	}`

	var msg BambuMessage
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if len(msg.Print.HMS) != 1 {
		t.Fatalf("expected 1 HMS error, got %d", len(msg.Print.HMS))
	}
	if msg.Print.HMS[0].Code != 100 {
		t.Errorf("expected HMS code 100, got %d", msg.Print.HMS[0].Code)
	}
	if msg.Print.HMS[0].Severity != 3 {
		t.Errorf("expected HMS severity 3, got %d", msg.Print.HMS[0].Severity)
	}
}

func TestBambuCommandSerialization_Pause(t *testing.T) {
	cmd := BambuCommand{
		Print: &BambuPrintCmd{
			Sequence: "1234567890",
			Command:  "pause",
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("failed to marshal command: %v", err)
	}

	expected := `{"print":{"sequence_id":"1234567890","command":"pause"}}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestBambuCommandSerialization_Resume(t *testing.T) {
	cmd := BambuCommand{
		Print: &BambuPrintCmd{
			Sequence: "1234567890",
			Command:  "resume",
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("failed to marshal command: %v", err)
	}

	expected := `{"print":{"sequence_id":"1234567890","command":"resume"}}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestBambuCommandSerialization_Stop(t *testing.T) {
	cmd := BambuCommand{
		Print: &BambuPrintCmd{
			Sequence: "1234567890",
			Command:  "stop",
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("failed to marshal command: %v", err)
	}

	expected := `{"print":{"sequence_id":"1234567890","command":"stop"}}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestBambuCommandSerialization_PushAll(t *testing.T) {
	cmd := BambuCommand{
		Pushing: &BambuPushingCmd{
			Sequence: "1234567890",
			Command:  "pushall",
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("failed to marshal command: %v", err)
	}

	expected := `{"pushing":{"sequence_id":"1234567890","command":"pushall"}}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestBambuCommandSerialization_LEDControl(t *testing.T) {
	cmd := BambuCommand{
		System: &BambuSystemCmd{
			Sequence: "1234567890",
			Command:  "ledctrl",
			LedMode:  "on",
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("failed to marshal command: %v", err)
	}

	expected := `{"system":{"sequence_id":"1234567890","command":"ledctrl","led_mode":"on"}}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestBambuCommandSerialization_PrintSpeed(t *testing.T) {
	cmd := BambuCommand{
		Print: &BambuPrintCmd{
			Sequence: "1234567890",
			Command:  "print_speed",
			Param:    strPtr("3"), // Sport mode
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("failed to marshal command: %v", err)
	}

	expected := `{"print":{"sequence_id":"1234567890","command":"print_speed","param":"3"}}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestBambuCommandSerialization_ProjectFile(t *testing.T) {
	cmd := BambuCommand{
		Print: &BambuPrintCmd{
			Sequence:    "1234567890",
			Command:     "project_file",
			ProjectID:   "0",
			ProfileID:   "0",
			TaskID:      "0",
			Subtask:     "0",
			SubtaskName: "test.3mf",
			URL:         "ftp:///cache/test.3mf",
			Filename:    "test.3mf",
			Timelapse:   true,
			BedLeveling: true,
			FlowCali:    false,
			Vibration:   false,
			AMS: &BambuAMSCmd{
				UseAMS:       true,
				PrintSpeed:   "standard",
				LayerInspect: false,
			},
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("failed to marshal command: %v", err)
	}

	// Verify key fields are present
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse marshaled data: %v", err)
	}

	printCmd, ok := parsed["print"].(map[string]interface{})
	if !ok {
		t.Fatal("expected print key in command")
	}
	if printCmd["command"] != "project_file" {
		t.Errorf("expected command 'project_file', got %v", printCmd["command"])
	}
	if printCmd["url"] != "ftp:///cache/test.3mf" {
		t.Errorf("expected url 'ftp:///cache/test.3mf', got %v", printCmd["url"])
	}
	if printCmd["timelapse"] != true {
		t.Errorf("expected timelapse true, got %v", printCmd["timelapse"])
	}
}

func TestGetStatus_ReturnsLastState(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.100", "12345678", "")

	// Simulate receiving a status update
	client.lastState = &model.PrinterState{
		PrinterID:   client.printerID,
		Status:      model.PrinterStatusPrinting,
		Progress:    75.0,
		CurrentFile: "test.3mf",
		BedTemp:     60.0,
		NozzleTemp:  210.0,
		UpdatedAt:   time.Now(),
	}

	state, err := client.GetStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Status != model.PrinterStatusPrinting {
		t.Errorf("expected status Printing, got %s", state.Status)
	}
	if state.Progress != 75.0 {
		t.Errorf("expected progress 75.0, got %f", state.Progress)
	}
}

func TestGetStatus_ReturnsOfflineWhenNoState(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.100", "12345678", "")
	client.lastState = nil

	state, err := client.GetStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Status != model.PrinterStatusOffline {
		t.Errorf("expected status Offline, got %s", state.Status)
	}
}

func TestBambuInfoParsing(t *testing.T) {
	jsonData := `{
		"info": {
			"sequence_id": "12345",
			"command": "get_version",
			"module": [
				{
					"name": "ota",
					"project_name": "N/A",
					"sw_ver": "01.06.00.00",
					"hw_ver": "AP05",
					"sn": "00M09A350100123"
				}
			]
		}
	}`

	var msg BambuMessage
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if msg.Info == nil {
		t.Fatal("expected info to be present")
	}
	if msg.Info.Command != "get_version" {
		t.Errorf("expected command 'get_version', got %s", msg.Info.Command)
	}
	if len(msg.Info.Module) != 1 {
		t.Fatalf("expected 1 module, got %d", len(msg.Info.Module))
	}
	if msg.Info.Module[0].SerNo != "00M09A350100123" {
		t.Errorf("expected serial '00M09A350100123', got %s", msg.Info.Module[0].SerNo)
	}
}

func TestBambuSystemParsing(t *testing.T) {
	jsonData := `{
		"system": {
			"sequence_id": "12345",
			"command": "ledctrl",
			"result": "success",
			"led_mode": "on"
		}
	}`

	var msg BambuMessage
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if msg.System == nil {
		t.Fatal("expected system to be present")
	}
	if msg.System.Result != "success" {
		t.Errorf("expected result 'success', got %s", msg.System.Result)
	}
	if msg.System.LedMode != "on" {
		t.Errorf("expected led_mode 'on', got %s", msg.System.LedMode)
	}
}

func TestBambuVTTrayParsing(t *testing.T) {
	jsonData := `{
		"print": {
			"gcode_state": "IDLE",
			"vt_tray": {
				"id": "254",
				"tray_id_name": "",
				"tray_type": "PLA",
				"tray_sub_brands": "",
				"tray_color": "FF0000FF",
				"tray_weight": "1000",
				"tray_diameter": "1.75",
				"tray_temp": "220",
				"bed_temp": "60",
				"nozzle_temp_max": "230",
				"nozzle_temp_min": "190",
				"remain": 95
			}
		}
	}`

	var msg BambuMessage
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if msg.Print.VTTray == nil {
		t.Fatal("expected vt_tray to be present")
	}
	if msg.Print.VTTray.TrayType != "PLA" {
		t.Errorf("expected tray type 'PLA', got %s", msg.Print.VTTray.TrayType)
	}
	if msg.Print.VTTray.TrayColor != "FF0000FF" {
		t.Errorf("expected tray color 'FF0000FF', got %s", msg.Print.VTTray.TrayColor)
	}
	if msg.Print.VTTray.Remain != 95 {
		t.Errorf("expected remain 95, got %d", msg.Print.VTTray.Remain)
	}
}

func TestBambuLightsReportParsing(t *testing.T) {
	jsonData := `{
		"print": {
			"gcode_state": "RUNNING",
			"lights_report": [
				{
					"node": "chamber_light",
					"mode": "on"
				},
				{
					"node": "work_light",
					"mode": "flashing"
				}
			]
		}
	}`

	var msg BambuMessage
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if len(msg.Print.LightsReport) != 2 {
		t.Fatalf("expected 2 lights, got %d", len(msg.Print.LightsReport))
	}
	if msg.Print.LightsReport[0].Node != "chamber_light" {
		t.Errorf("expected node 'chamber_light', got %s", msg.Print.LightsReport[0].Node)
	}
	if msg.Print.LightsReport[0].Mode != "on" {
		t.Errorf("expected mode 'on', got %s", msg.Print.LightsReport[0].Mode)
	}
}

type fakeToken struct {
	err   error
	codes map[string]byte
}

func (f fakeToken) Wait() bool                     { return true }
func (f fakeToken) WaitTimeout(time.Duration) bool { return true }
func (f fakeToken) Error() error                   { return f.err }

var errToken = errors.New("broker rejected publish")

func (f fakeToken) Result() map[string]byte { return f.codes }
func (f fakeToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// fakeMessage is a minimal mqtt.Message for handler wiring tests.
type fakeMessage struct {
	topic   string
	payload []byte
}

func (m fakeMessage) Duplicate() bool   { return false }
func (m fakeMessage) Qos() byte         { return 0 }
func (m fakeMessage) Retained() bool    { return false }
func (m fakeMessage) Topic() string     { return m.topic }
func (m fakeMessage) MessageID() uint16 { return 1 }
func (m fakeMessage) Payload() []byte   { return m.payload }
func (m fakeMessage) Ack()              {}

type recordedPub struct {
	topic   string
	qos     byte
	payload []byte
}

// fakeMQTT records subscribe and publish calls from onConnect.
type fakeMQTT struct {
	mqtt.Client
	mu       sync.Mutex
	subs     []string
	handlers []mqtt.MessageHandler
	pubs     []recordedPub
	subCode  byte
	pubErr   error
}

func (f *fakeMQTT) Subscribe(topic string, _ byte, cb mqtt.MessageHandler) mqtt.Token {
	f.mu.Lock()
	f.subs = append(f.subs, topic)
	f.handlers = append(f.handlers, cb)
	code := f.subCode
	f.mu.Unlock()
	return fakeToken{codes: map[string]byte{topic: code}}
}

func (f *fakeMQTT) Publish(topic string, qos byte, _ bool, payload interface{}) mqtt.Token {
	var body []byte
	switch p := payload.(type) {
	case []byte:
		body = append([]byte(nil), p...)
	case string:
		body = []byte(p)
	}
	f.mu.Lock()
	f.pubs = append(f.pubs, recordedPub{topic: topic, qos: qos, payload: body})
	err := f.pubErr
	f.mu.Unlock()
	return fakeToken{err: err}
}

func (f *fakeMQTT) snapshot() (subs []string, pubs []recordedPub) {
	f.mu.Lock()
	defer f.mu.Unlock()
	subs = append([]string(nil), f.subs...)
	pubs = append([]recordedPub(nil), f.pubs...)
	return subs, pubs
}

func stopBambuLoops(c *BambuClient) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mqttClient = nil
	if c.stopChan != nil {
		close(c.stopChan)
		c.stopChan = nil
	}
}

// syncBuffer is an io.Writer safe to read while slog writes from another goroutine.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func captureLogs(t *testing.T) *syncBuffer {
	t.Helper()
	buf := &syncBuffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

func TestOnConnect_PublishesPushAllAfterSubscribeAndOnReconnect(t *testing.T) {
	serial := "22E8AA000000001"
	client := NewBambuClient(uuid.New(), "192.168.1.50", "12345678", serial)
	client.pushAllInterval = time.Hour
	client.reportTimeout = time.Hour
	t.Cleanup(func() { stopBambuLoops(client) })

	fake := &fakeMQTT{}
	client.onConnect(fake)
	client.onConnect(fake)

	subs, pubs := fake.snapshot()
	if len(subs) != 2 || subs[0] != "device/"+serial+"/report" || subs[1] != "device/"+serial+"/report" {
		t.Fatalf("subscriptions = %#v", subs)
	}
	fake.mu.Lock()
	if len(fake.handlers) != 2 || fake.handlers[0] == nil || fake.handlers[1] == nil {
		t.Fatal("subscribe handler was not wired")
	}
	handler := fake.handlers[0]
	fake.mu.Unlock()

	const wantPush = `{"pushing":{"sequence_id":"0","command":"pushall"}}`
	const wantVersion = `{"info":{"sequence_id":"0","command":"get_version"}}`
	pushN, verN := 0, 0
	for i, pub := range pubs {
		if pub.topic != "device/"+serial+"/request" {
			t.Errorf("pub %d topic = %s", i, pub.topic)
		}
		if pub.qos != 0 {
			t.Errorf("pub %d qos = %d, want 0", i, pub.qos)
		}
		switch string(pub.payload) {
		case wantPush:
			pushN++
		case wantVersion:
			verN++
		default:
			t.Errorf("pub %d payload = %s", i, pub.payload)
		}
	}
	if pushN != 2 || verN != 2 {
		t.Fatalf("pushall=%d get_version=%d, pubs=%d", pushN, verN, len(pubs))
	}

	handler(nil, fakeMessage{
		topic:   "device/" + serial + "/report",
		payload: []byte(`{"print":{"bed_temper":30}}`),
	})
	state, _ := client.GetStatus()
	if state.Status == model.PrinterStatusOffline {
		t.Fatal("subscribe handler did not apply the report")
	}
}

func TestPushLoop_PublishesAgain(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.50", "12345678", "22E8AA000000001")
	client.pushAllInterval = 15 * time.Millisecond
	client.reportTimeout = time.Hour
	t.Cleanup(func() { stopBambuLoops(client) })

	fake := &fakeMQTT{}
	client.onConnect(fake)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		_, pubs := fake.snapshot()
		// onConnect publishes get_version and pushall; the loop adds another pushall.
		if len(pubs) >= 3 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	_, pubs := fake.snapshot()
	t.Fatalf("expected a periodic pushall, got %d publishes", len(pubs))
}

func TestNewBambuClientID_IsRandomAndMQTT31Safe(t *testing.T) {
	a := newBambuClientID()
	b := newBambuClientID()
	if a == b {
		t.Fatalf("client ids collided: %s", a)
	}
	for _, id := range []string{a, b} {
		if len(id) > 23 {
			t.Fatalf("client id %q is %d bytes; MQTT 3.1 allows 23", id, len(id))
		}
		if !strings.HasPrefix(id, "rankos-") {
			t.Fatalf("client id %q", id)
		}
		if strings.Contains(id, "printfarm") {
			t.Fatalf("client id still uses the stable prefix: %s", id)
		}
	}
}

func TestMQTTOptions_MatchWorkingLANClient(t *testing.T) {
	c := NewBambuClient(uuid.New(), "192.168.1.50", "12345678", "22E8AA000000001")
	tlsCfg := &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}
	opts := c.mqttOptions("ssl://192.168.1.50:8883", "rankos-0011223344556677", "bblp", "12345678", tlsCfg)
	if opts.ClientID != "rankos-0011223344556677" {
		t.Fatalf("client id %s", opts.ClientID)
	}
	if opts.ProtocolVersion != 4 {
		t.Fatalf("protocol %d, want MQTT 3.1.1 (4)", opts.ProtocolVersion)
	}
	if !opts.CleanSession {
		t.Fatal("clean session should be set")
	}
	if opts.Order {
		t.Fatal("message order should not be required; a slow handler must not stall the session")
	}
	if opts.TLSConfig == nil || !opts.TLSConfig.InsecureSkipVerify {
		t.Fatal("LAN TLS must skip verification for the printer's self-signed cert")
	}
	if opts.Username != "bblp" {
		t.Fatalf("username %s", opts.Username)
	}
}

func TestSubscribe_RefusedSUBACKIsLogged(t *testing.T) {
	buf := captureLogs(t)
	client := NewBambuClient(uuid.New(), "192.168.1.50", "12345678", "22E8AA000000001")
	client.pushAllInterval = time.Hour
	client.reportTimeout = time.Hour
	t.Cleanup(func() { stopBambuLoops(client) })

	fake := &fakeMQTT{subCode: 0x80}
	client.onConnect(fake)

	logs := buf.String()
	if !strings.Contains(logs, "0x80") || !strings.Contains(logs, "subscription refused") {
		t.Fatalf("missing SUBACK refusal, logs: %s", logs)
	}
	_, pubs := fake.snapshot()
	if len(pubs) == 0 {
		t.Fatal("pushall was skipped after a refused subscription")
	}
}

func TestPublish_TokenErrorIsLogged(t *testing.T) {
	buf := captureLogs(t)
	client := NewBambuClient(uuid.New(), "192.168.1.50", "12345678", "22E8AA000000001")
	client.pushAllInterval = time.Hour
	client.reportTimeout = time.Hour
	t.Cleanup(func() { stopBambuLoops(client) })

	fake := &fakeMQTT{pubErr: errToken}
	client.onConnect(fake)

	logs := buf.String()
	if !strings.Contains(logs, "token_error") || !strings.Contains(logs, "broker rejected") {
		t.Fatalf("missing publish token error, logs: %s", logs)
	}
}

func TestDebugFlag_SubscribesWildcardAndLogsPayload(t *testing.T) {
	t.Setenv("BAMBU_MQTT_DEBUG", "1")
	buf := captureLogs(t)
	client := NewBambuClient(uuid.New(), "192.168.1.50", "12345678", "22E8AA000000001")
	client.pushAllInterval = time.Hour
	client.reportTimeout = time.Hour
	t.Cleanup(func() { stopBambuLoops(client) })

	fake := &fakeMQTT{}
	client.onConnect(fake)
	subs, _ := fake.snapshot()
	if len(subs) != 2 || subs[1] != "device/+/report" {
		t.Fatalf("subscriptions = %#v", subs)
	}
	fake.mu.Lock()
	debugHandler := fake.handlers[1]
	fake.mu.Unlock()
	long := strings.Repeat("x", 400)
	debugHandler(nil, fakeMessage{topic: "device/OTHERSERIAL/report", payload: []byte(long)})

	logs := buf.String()
	if !strings.Contains(logs, "device/OTHERSERIAL/report") {
		t.Fatalf("debug log missing topic: %s", logs)
	}
	if !strings.Contains(logs, strings.Repeat("x", 300)) {
		t.Fatal("debug log did not include the first 300 bytes")
	}
	if strings.Contains(logs, strings.Repeat("x", 301)) {
		t.Fatal("debug log included more than 300 bytes")
	}
}

func TestConnectionLost_LogsReason(t *testing.T) {
	buf := captureLogs(t)
	client := NewBambuClient(uuid.New(), "192.168.1.50", "12345678", "22E8AA000000001")
	client.clientID = "rankos-0011223344556677"
	client.onConnectionLost(nil, errToken)
	logs := buf.String()
	if !strings.Contains(logs, "connection lost") || !strings.Contains(logs, "broker rejected") {
		t.Fatalf("missing disconnect reason, logs: %s", logs)
	}
	if !strings.Contains(logs, "rankos-0011223344556677") {
		t.Fatalf("missing client id, logs: %s", logs)
	}
}

func TestNoReport_PublishesStart(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.50", "12345678", "22E8AA000000001")
	client.reportTimeout = 15 * time.Millisecond
	fake := &fakeMQTT{}
	client.mu.Lock()
	client.mqttClient = fake
	client.connected = true
	client.mu.Unlock()
	t.Cleanup(func() { stopBambuLoops(client) })

	client.armNoReportWatch()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		_, pubs := fake.snapshot()
		for _, pub := range pubs {
			if string(pub.payload) == `{"pushing":{"sequence_id":"0","command":"start"}}` {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	_, pubs := fake.snapshot()
	t.Fatalf("expected start after silence, pubs=%v", pubs)
}

func TestNoReport_WarnsToCheckSerial(t *testing.T) {
	buf := captureLogs(t)
	client := NewBambuClient(uuid.New(), "192.168.1.50", "12345678", "22E8AA000000001")
	client.reportTimeout = 20 * time.Millisecond
	t.Cleanup(func() { stopBambuLoops(client) })

	client.armNoReportWatch()
	time.Sleep(60 * time.Millisecond)

	logs := buf.String()
	if !strings.Contains(logs, "no reports received; check that the serial matches the printer") {
		t.Fatalf("missing serial warning, logs: %s", logs)
	}
	if !strings.Contains(logs, "22E8AA000000001") {
		t.Fatalf("warning missing serial, logs: %s", logs)
	}
}

func TestReport_SuppressesWarningAndLeavesOffline(t *testing.T) {
	buf := captureLogs(t)
	client := NewBambuClient(uuid.New(), "192.168.1.50", "12345678", "22E8AA000000001")
	client.reportTimeout = 40 * time.Millisecond
	t.Cleanup(func() { stopBambuLoops(client) })

	client.armNoReportWatch()
	client.ingestReport("device/22E8AA000000001/report", []byte(`{"print":{"bed_temper":21.5}}`))
	time.Sleep(70 * time.Millisecond)

	logs := buf.String()
	if strings.Contains(logs, "no reports received") {
		t.Fatalf("warning fired after a report: %s", logs)
	}
	if !strings.Contains(logs, "first Bambu report received") {
		t.Fatalf("missing debug log for first report: %s", logs)
	}
	state, err := client.GetStatus()
	if err != nil {
		t.Fatal(err)
	}
	if state.Status == model.PrinterStatusOffline {
		t.Fatal("status still offline after a report")
	}
	if state.Status != model.PrinterStatusIdle {
		t.Fatalf("status = %s, want idle", state.Status)
	}
	if state.BedTemp != 21.5 {
		t.Fatalf("bed temp = %v", state.BedTemp)
	}
}

func TestIngestReport_AcceptsNumericStrings(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.50", "12345678", "00M00A000000001")
	client.ingestReport("device/00M00A000000001/report", []byte(`{
		"print": {
			"gcode_state": "IDLE",
			"cooling_fan_speed": "15",
			"bed_temper": "25.5"
		}
	}`))
	state, _ := client.GetStatus()
	if state.Status != model.PrinterStatusIdle {
		t.Fatalf("status = %s", state.Status)
	}
	if state.BedTemp != 25.5 {
		t.Fatalf("bed = %v", state.BedTemp)
	}
	if state.CoolingFanSpeed != 100 {
		t.Fatalf("fan = %d, want 100 from raw 15", state.CoolingFanSpeed)
	}
	client.ingestReport("device/00M00A000000001/report", []byte(`{"print":{"nozzle_temper":"200"}}`))
	state, _ = client.GetStatus()
	if state.BedTemp != 25.5 || state.NozzleTemp != 200 || state.CoolingFanSpeed != 100 {
		t.Fatalf("delta wiped state: bed=%v nozzle=%v fan=%d", state.BedTemp, state.NozzleTemp, state.CoolingFanSpeed)
	}
}

func TestIngestReport_MergesDelta(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.168.1.50", "12345678", "22E8AA000000001")
	client.ingestReport("device/22E8AA000000001/report", []byte(`{
		"print": {
			"gcode_state": "RUNNING",
			"mc_percent": 40,
			"bed_temper": 60,
			"nozzle_temper": 200,
			"subtask_name": "part.3mf"
		}
	}`))
	client.ingestReport("device/22E8AA000000001/report", []byte(`{"print":{"nozzle_temper":210}}`))

	state, _ := client.GetStatus()
	if state.Status != model.PrinterStatusPrinting {
		t.Fatalf("status = %s, want printing", state.Status)
	}
	if state.Progress != 40 {
		t.Fatalf("progress = %v", state.Progress)
	}
	if state.BedTemp != 60 {
		t.Fatalf("bed = %v", state.BedTemp)
	}
	if state.NozzleTemp != 210 {
		t.Fatalf("nozzle = %v", state.NozzleTemp)
	}
	if state.CurrentFile != "part.3mf" {
		t.Fatalf("file = %q", state.CurrentFile)
	}

	// Explicit zeros in a delta must apply without clearing fields that were omitted.
	client.ingestReport("device/22E8AA000000001/report", []byte(`{"print":{"bed_temper":0,"mc_percent":0}}`))
	state, _ = client.GetStatus()
	if state.BedTemp != 0 {
		t.Fatalf("bed temp not cleared, got %v", state.BedTemp)
	}
	if state.Progress != 0 {
		t.Fatalf("progress not cleared, got %v", state.Progress)
	}
	if state.NozzleTemp != 210 {
		t.Fatalf("nozzle wiped by delta, got %v", state.NozzleTemp)
	}
	if state.Status != model.PrinterStatusPrinting {
		t.Fatalf("status wiped by delta, got %s", state.Status)
	}
	if state.CurrentFile != "part.3mf" {
		t.Fatalf("file wiped by delta, got %q", state.CurrentFile)
	}
}

// BenchmarkParsePrintStatus benchmarks status parsing performance.
func BenchmarkParsePrintStatus(b *testing.B) {
	client := NewBambuClient(uuid.New(), "192.168.1.100", "12345678", "")

	printStatus := &BambuPrintStatus{
		GcodeState:         "RUNNING",
		MCPercent:          45,
		MCRemainingTime:    120,
		SubtaskName:        "test_print.3mf",
		BedTemper:          60.0,
		BedTargetTemper:    60.0,
		NozzleTemper:       210.0,
		NozzleTargetTemper: 215.0,
		LayerNum:           100,
		TotalLayerNum:      200,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.mergePrintStatus(printStatus)
	}
}

// BenchmarkBambuMessageUnmarshal benchmarks message parsing performance.
func BenchmarkBambuMessageUnmarshal(b *testing.B) {
	jsonData := []byte(`{
		"print": {
			"gcode_state": "RUNNING",
			"mc_percent": 75,
			"mc_remaining_time": 30,
			"subtask_name": "benchy.3mf",
			"bed_temper": 60.5,
			"nozzle_temper": 215.2,
			"layer_num": 150,
			"total_layer_num": 200
		}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var msg BambuMessage
		json.Unmarshal(jsonData, &msg)
	}
}
