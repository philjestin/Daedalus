package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ReportStaleAfter is how long a printer may stay silent before it is offline.
// The card plan uses 30 seconds without an MQTT report.
const ReportStaleAfter = 30 * time.Second

// ViewState is the permission state for printer controls.
// Status stays the operational state (idle, printing, paused, error, offline).
// View is what the server will actually accept.
type ViewState string

const (
	ViewOffline  ViewState = "offline"
	ViewReadOnly ViewState = "read_only"
	ViewIdle     ViewState = "idle"
	ViewPrinting ViewState = "printing"
	ViewPaused   ViewState = "paused"
	ViewError    ViewState = "error"
	ViewFinished ViewState = "finished"
)

// CameraAvailability is how the card should treat the printer camera.
// It comes from the MQTT ipcam report. Streaming itself is a later change.
type CameraAvailability string

const (
	CameraAvailable    CameraAvailability = "available"
	CameraLiveviewOff  CameraAvailability = "liveview_off"
	CameraNotSupported CameraAvailability = "not_supported"
	CameraUnknown      CameraAvailability = "unknown"
)

// CameraInfo is the camera field on PrinterState.
type CameraInfo struct {
	Availability CameraAvailability `json:"availability"`
}

// IPCamReport is the part of an MQTT ipcam object that decides availability.
// The host inside RTSPURL is ignored. RankOS uses the printer's configured address.
type IPCamReport struct {
	Present       bool
	RTSPURL       string
	LiveviewLocal string
}

// Capabilities is what this printer can do, from its family and from the report.
// DevMode false means the firmware requires a signed command, so View is read_only.
// Supported, when non-nil, is the protocol allow-list (OctoPrint and Moonraker
// only pause, resume, and stop). Nil means every command the view allows.
type Capabilities struct {
	DevMode       bool `json:"dev_mode"`
	ChamberHeater bool `json:"chamber_heater"`
	Nozzles       int  `json:"nozzles"`
	SecondLight   bool `json:"second_light"`
	AuxFan        bool `json:"aux_fan"`
	ChamberFan    bool `json:"chamber_fan"`
	AirductModes  bool `json:"airduct_modes"`
	// MQTTBedTemp is true when the printer should be sent print.set_bed_temp
	// instead of M140. Left false until a report actually announces it.
	MQTTBedTemp bool          `json:"mqtt_bed_temp"`
	Supported   []CommandKind `json:"supported,omitempty"`
}

// CommandKind is one action in the closed command union.
type CommandKind string

const (
	CmdPause       CommandKind = "pause"
	CmdResume      CommandKind = "resume"
	CmdStop        CommandKind = "stop"
	CmdSetSpeed    CommandKind = "set_speed"
	CmdSetLight    CommandKind = "set_light"
	CmdSetTemp     CommandKind = "set_temp"
	CmdSetFan      CommandKind = "set_fan"
	CmdJog         CommandKind = "jog"
	CmdHome        CommandKind = "home"
	CmdExtrude     CommandKind = "extrude"
	CmdAMSLoad     CommandKind = "ams_load"
	CmdAMSUnload   CommandKind = "ams_unload"
	CmdAMSRFID     CommandKind = "ams_rfid"
	CmdSkipObjects CommandKind = "skip_objects"
	CmdCalibrate   CommandKind = "calibrate"
	CmdClearError  CommandKind = "clear_error"
	CmdStartPrint  CommandKind = "start_print"
)

// Light nodes accepted by set_light. OpenBambuAPI names chamber_light and
// work_light. H2 printers also report chamber_light2.
const (
	LightChamber  = "chamber_light"
	LightChamber2 = "chamber_light2"
	LightWork     = "work_light"
)

// HMS severities from ha-bambulab. Lower is worse.
const (
	HMSFatal   = 1
	HMSSerious = 2
)

// ExtrudeMinNozzleC is the lowest nozzle temperature that may extrude.
const ExtrudeMinNozzleC = 170

// ErrCommandNotAllowed means the printer's current view refuses the command.
var ErrCommandNotAllowed = errors.New("command is not allowed in the current printer state")

// ErrUnsupported means this connection cannot carry the command.
var ErrUnsupported = errors.New("command is not supported by this printer")

// ErrPrinterNotFound means the id is not in the database.
var ErrPrinterNotFound = errors.New("printer not found")

// CommandParseError is a bad command body. The API returns 400.
type CommandParseError struct {
	Reason string
}

func (e *CommandParseError) Error() string {
	if e == nil || e.Reason == "" {
		return "invalid command"
	}
	return e.Reason
}

// Command is the closed set of printer actions. The unexported method keeps
// the implementations in this package, so the web cannot invent a new one.
type Command interface {
	kind() CommandKind
}

// Pause pauses whatever is printing, including a job started outside RankOS.
type Pause struct{}

func (Pause) kind() CommandKind { return CmdPause }

// Resume resumes a paused print.
type Resume struct{}

func (Resume) kind() CommandKind { return CmdResume }

// Stop stops the current print. It cannot be undone.
type Stop struct{}

func (Stop) kind() CommandKind { return CmdStop }

// SetSpeed selects a print speed level from 1 (silent) to 4 (ludicrous).
type SetSpeed struct {
	Level int
}

func (SetSpeed) kind() CommandKind { return CmdSetSpeed }

// SetLight turns a chamber or work light on or off.
type SetLight struct {
	Node string
	On   bool
}

func (SetLight) kind() CommandKind { return CmdSetLight }

// CommandKindOf returns the command's kind.
func CommandKindOf(c Command) CommandKind {
	if c == nil {
		return ""
	}
	return c.kind()
}

// commandKinds is the stable order used in PrinterState.allowed.
var commandKinds = []CommandKind{
	CmdPause,
	CmdResume,
	CmdStop,
	CmdSetSpeed,
	CmdSetLight,
	CmdSetTemp,
	CmdSetFan,
	CmdJog,
	CmdHome,
	CmdExtrude,
	CmdAMSLoad,
	CmdAMSUnload,
	CmdAMSRFID,
	CmdSkipObjects,
	CmdCalibrate,
	CmdClearError,
	CmdStartPrint,
}

// DeriveView computes the permission state. It is a pure function.
// Offline wins, then read-only (no Developer Mode), then error, then the
// gcode state. FINISH and FAILED with no error code are finished, so the
// piece is still on the bed. FAILED with a print error is error.
func DeriveView(state *PrinterState, now time.Time) ViewState {
	if state == nil {
		return ViewOffline
	}
	if state.Status == PrinterStatusOffline {
		return ViewOffline
	}
	if !state.UpdatedAt.IsZero() && now.Sub(state.UpdatedAt) > ReportStaleAfter {
		return ViewOffline
	}
	if !state.Caps.DevMode {
		return ViewReadOnly
	}
	if state.PrintError != 0 || severeHMS(state.HMSErrors) {
		return ViewError
	}
	switch strings.ToUpper(state.GcodeState) {
	case "FINISH", "FAILED":
		return ViewFinished
	case "PAUSE":
		return ViewPaused
	case "RUNNING", "PREPARE":
		return ViewPrinting
	case "IDLE":
		return ViewIdle
	}
	switch state.Status {
	case PrinterStatusPrinting:
		return ViewPrinting
	case PrinterStatusPaused:
		return ViewPaused
	case PrinterStatusError:
		return ViewError
	default:
		return ViewIdle
	}
}

func severeHMS(errors []HMSError) bool {
	for _, h := range errors {
		if h.Severity == HMSFatal || h.Severity == HMSSerious {
			return true
		}
	}
	return false
}

// RefreshView stores View, FinishAt, and Allowed on state.
func RefreshView(state *PrinterState, now time.Time) {
	if state == nil {
		return
	}
	state.View = DeriveView(state, now)
	state.FinishAt = finishAt(state)
	state.Allowed = AllowedCommands(state)
}

// finishAt is UpdatedAt plus the remaining seconds, only while a print is
// running or paused. TimeLeft is already in seconds.
func finishAt(state *PrinterState) *time.Time {
	if state.TimeLeft <= 0 {
		return nil
	}
	if state.View != ViewPrinting && state.View != ViewPaused {
		return nil
	}
	base := state.UpdatedAt
	if base.IsZero() {
		return nil
	}
	t := base.Add(time.Duration(state.TimeLeft) * time.Second)
	return &t
}

// AllowedCommands is the permission table. A command is listed only when the
// handler will accept it. Locked and hidden controls are omitted.
func AllowedCommands(state *PrinterState) []CommandKind {
	if state == nil {
		return []CommandKind{}
	}
	view := state.View
	if view == "" {
		view = DeriveView(state, time.Now())
	}
	out := make([]CommandKind, 0)
	for _, kind := range commandKinds {
		if !viewAllows(view, kind, state) {
			continue
		}
		if !protocolAllows(state.Caps, kind) {
			continue
		}
		out = append(out, kind)
	}
	return out
}

func protocolAllows(caps Capabilities, kind CommandKind) bool {
	if caps.Supported == nil {
		return true
	}
	for _, have := range caps.Supported {
		if have == kind {
			return true
		}
	}
	return false
}

func viewAllows(view ViewState, kind CommandKind, state *PrinterState) bool {
	switch kind {
	case CmdPause:
		return view == ViewPrinting
	case CmdResume:
		if view == ViewPaused {
			return true
		}
		return view == ViewError && pausedBeforeError(state)
	case CmdStop:
		return view == ViewPrinting || view == ViewPaused || view == ViewError
	case CmdSetSpeed:
		return view == ViewPrinting || view == ViewPaused
	case CmdSetLight:
		switch view {
		case ViewIdle, ViewFinished, ViewPrinting, ViewPaused, ViewError:
			return true
		default:
			return false
		}
	case CmdSetTemp, CmdSetFan:
		switch view {
		case ViewIdle, ViewFinished, ViewPrinting, ViewPaused, ViewError:
			return true
		default:
			return false
		}
	case CmdJog, CmdHome:
		return view == ViewIdle || view == ViewFinished
	case CmdExtrude:
		if view != ViewIdle && view != ViewFinished {
			return false
		}
		return state.NozzleTemp >= ExtrudeMinNozzleC
	case CmdAMSLoad, CmdAMSUnload:
		return view == ViewIdle || view == ViewFinished || view == ViewPaused
	case CmdAMSRFID:
		switch view {
		case ViewIdle, ViewFinished, ViewPrinting, ViewPaused, ViewError:
			return true
		default:
			return false
		}
	case CmdSkipObjects:
		return view == ViewPrinting || view == ViewPaused
	case CmdCalibrate, CmdStartPrint:
		return view == ViewIdle
	case CmdClearError:
		return view == ViewError
	default:
		return false
	}
}

func pausedBeforeError(state *PrinterState) bool {
	if state == nil {
		return false
	}
	if strings.EqualFold(state.GcodeState, "PAUSE") {
		return true
	}
	return state.Status == PrinterStatusPaused
}

// CommandAllowed reports whether cmd may be sent for this state.
func CommandAllowed(state *PrinterState, cmd Command) bool {
	if state == nil || cmd == nil {
		return false
	}
	kind := cmd.kind()
	for _, have := range AllowedCommands(state) {
		if have == kind {
			return true
		}
	}
	return false
}

// DeriveCamera applies the ipcam rules. bambuLAN is false for cloud, OctoPrint,
// Moonraker, and manual printers.
func DeriveCamera(bambuLAN bool, report IPCamReport) CameraAvailability {
	if !bambuLAN {
		return CameraNotSupported
	}
	if !report.Present {
		return CameraUnknown
	}
	rtsp := strings.TrimSpace(report.RTSPURL)
	local := strings.ToLower(strings.TrimSpace(report.LiveviewLocal))
	// rtsp_url wins when the printer sent one, same as Bambu Studio.
	if rtsp != "" {
		switch {
		case strings.EqualFold(rtsp, "disable"):
			return CameraLiveviewOff
		case strings.HasPrefix(strings.ToLower(rtsp), "rtsps://"):
			return CameraAvailable
		default:
			return CameraNotSupported
		}
	}
	switch local {
	case "":
		return CameraUnknown
	case "disabled":
		return CameraLiveviewOff
	case "rtsps":
		return CameraAvailable
	default:
		// "local" is the still-image port used by A1 and P1. Anything else
		// is also not a stream this server can open yet.
		return CameraNotSupported
	}
}

type commandBody struct {
	Kind  string  `json:"kind"`
	Level *int    `json:"level"`
	Node  *string `json:"node"`
	On    *bool   `json:"on"`
}

// ParseCommand turns a request body into a Command.
// Only pause, resume, stop, set_speed, and set_light are accepted.
// A free gcode field is rejected.
func ParseCommand(data []byte) (Command, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, &CommandParseError{Reason: "command body is empty"}
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	var body commandBody
	if err := dec.Decode(&body); err != nil {
		return nil, &CommandParseError{Reason: "invalid command"}
	}
	if dec.More() {
		return nil, &CommandParseError{Reason: "invalid command"}
	}
	kind := CommandKind(strings.TrimSpace(body.Kind))
	switch kind {
	case CmdPause, CmdResume, CmdStop:
		if body.Level != nil || body.Node != nil || body.On != nil {
			return nil, &CommandParseError{Reason: "pause, resume, and stop take no extra fields"}
		}
		switch kind {
		case CmdPause:
			return Pause{}, nil
		case CmdResume:
			return Resume{}, nil
		default:
			return Stop{}, nil
		}
	case CmdSetSpeed:
		if body.Node != nil || body.On != nil {
			return nil, &CommandParseError{Reason: "set_speed only accepts level"}
		}
		if body.Level == nil {
			return nil, &CommandParseError{Reason: "set_speed requires level"}
		}
		if *body.Level < 1 || *body.Level > 4 {
			return nil, &CommandParseError{Reason: "speed level must be from 1 to 4"}
		}
		return SetSpeed{Level: *body.Level}, nil
	case CmdSetLight:
		if body.Level != nil {
			return nil, &CommandParseError{Reason: "set_light only accepts node and on"}
		}
		if body.Node == nil || strings.TrimSpace(*body.Node) == "" {
			return nil, &CommandParseError{Reason: "set_light requires node"}
		}
		if body.On == nil {
			return nil, &CommandParseError{Reason: "set_light requires on"}
		}
		node := strings.TrimSpace(*body.Node)
		switch node {
		case LightChamber, LightChamber2, LightWork:
		default:
			return nil, &CommandParseError{Reason: "light node must be chamber_light, chamber_light2, or work_light"}
		}
		return SetLight{Node: node, On: *body.On}, nil
	case "gcode", "gcode_line":
		return nil, &CommandParseError{Reason: "raw gcode is not accepted"}
	default:
		if kind == "" {
			return nil, &CommandParseError{Reason: "command kind is required"}
		}
		return nil, &CommandParseError{Reason: fmt.Sprintf("unknown command %q", kind)}
	}
}
