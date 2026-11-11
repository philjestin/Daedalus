package model

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func devState(status PrinterStatus, gcode string) *PrinterState {
	return &PrinterState{
		Status:     status,
		GcodeState: gcode,
		UpdatedAt:  time.Now(),
		Caps:       Capabilities{DevMode: true, Nozzles: 1},
		Camera:     CameraInfo{Availability: CameraUnknown},
	}
}

func TestDeriveView(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		edit func(*PrinterState)
		want ViewState
	}{
		{name: "offline", edit: func(s *PrinterState) { s.Status = PrinterStatusOffline }, want: ViewOffline},
		{
			name: "silent for more than 30s",
			edit: func(s *PrinterState) { s.UpdatedAt = now.Add(-31 * time.Second) },
			want: ViewOffline,
		},
		{
			name: "report 29s ago stays idle",
			edit: func(s *PrinterState) { s.UpdatedAt = now.Add(-29 * time.Second) },
			want: ViewIdle,
		},
		{
			name: "no developer mode is read only even while printing",
			edit: func(s *PrinterState) {
				s.Status = PrinterStatusPrinting
				s.GcodeState = "RUNNING"
				s.Caps.DevMode = false
			},
			want: ViewReadOnly,
		},
		{
			name: "offline wins over missing developer mode",
			edit: func(s *PrinterState) {
				s.Status = PrinterStatusOffline
				s.Caps.DevMode = false
			},
			want: ViewOffline,
		},
		{
			name: "print error",
			edit: func(s *PrinterState) {
				s.Status = PrinterStatusPrinting
				s.GcodeState = "RUNNING"
				s.PrintError = 0x05004001
			},
			want: ViewError,
		},
		{
			name: "serious hms",
			edit: func(s *PrinterState) {
				s.HMSErrors = []HMSError{{Severity: HMSSerious, Code: 1}}
			},
			want: ViewError,
		},
		{
			name: "info hms stays idle",
			edit: func(s *PrinterState) {
				s.HMSErrors = []HMSError{{Severity: 4, Code: 1}}
			},
			want: ViewIdle,
		},
		{
			name: "finish is finished while status stays idle",
			edit: func(s *PrinterState) {
				s.Status = PrinterStatusIdle
				s.GcodeState = "FINISH"
			},
			want: ViewFinished,
		},
		{
			name: "failed without an error code is finished",
			edit: func(s *PrinterState) { s.GcodeState = "FAILED" },
			want: ViewFinished,
		},
		{
			name: "failed with an error code is error",
			edit: func(s *PrinterState) {
				s.Status = PrinterStatusError
				s.GcodeState = "FAILED"
				s.PrintError = 1
			},
			want: ViewError,
		},
		{name: "pause", edit: func(s *PrinterState) {
			s.Status = PrinterStatusPaused
			s.GcodeState = "PAUSE"
		}, want: ViewPaused},
		{name: "running", edit: func(s *PrinterState) {
			s.Status = PrinterStatusPrinting
			s.GcodeState = "RUNNING"
		}, want: ViewPrinting},
		{name: "prepare", edit: func(s *PrinterState) {
			s.Status = PrinterStatusPrinting
			s.GcodeState = "PREPARE"
		}, want: ViewPrinting},
		{name: "idle", edit: func(s *PrinterState) { s.GcodeState = "IDLE" }, want: ViewIdle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := devState(PrinterStatusIdle, "IDLE")
			tt.edit(s)
			if got := DeriveView(s, now); got != tt.want {
				t.Fatalf("DeriveView() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDeriveView_zeroTimeIsNotStale(t *testing.T) {
	s := devState(PrinterStatusPrinting, "RUNNING")
	s.UpdatedAt = time.Time{}
	if got := DeriveView(s, time.Now()); got != ViewPrinting {
		t.Fatalf("got %s", got)
	}
}

func TestRefreshView_finishAt(t *testing.T) {
	updated := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	s := devState(PrinterStatusPrinting, "RUNNING")
	s.UpdatedAt = updated
	s.TimeLeft = 90 * 60
	RefreshView(s, updated)
	if s.View != ViewPrinting {
		t.Fatalf("view %s", s.View)
	}
	if s.FinishAt == nil || !s.FinishAt.Equal(updated.Add(90*time.Minute)) {
		t.Fatalf("finish_at = %v", s.FinishAt)
	}
	s.GcodeState = "IDLE"
	s.Status = PrinterStatusIdle
	s.TimeLeft = 0
	RefreshView(s, updated)
	if s.FinishAt != nil {
		t.Fatalf("idle finish_at = %v", s.FinishAt)
	}
}

func TestDeriveCamera(t *testing.T) {
	tests := []struct {
		name string
		lan  bool
		rep  IPCamReport
		want CameraAvailability
	}{
		{name: "not bambu lan", lan: false, rep: IPCamReport{Present: true, RTSPURL: "rtsps://printer/streaming/live/1"}, want: CameraNotSupported},
		{name: "no ipcam yet", lan: true, want: CameraUnknown},
		{name: "rtsp disable", lan: true, rep: IPCamReport{Present: true, RTSPURL: "disable", LiveviewLocal: "rtsps"}, want: CameraLiveviewOff},
		{name: "rtsps url wins", lan: true, rep: IPCamReport{Present: true, RTSPURL: "rtsps://printer/streaming/live/1", LiveviewLocal: "disabled"}, want: CameraAvailable},
		{name: "liveview disabled", lan: true, rep: IPCamReport{Present: true, LiveviewLocal: "disabled"}, want: CameraLiveviewOff},
		{name: "liveview rtsps", lan: true, rep: IPCamReport{Present: true, LiveviewLocal: "rtsps"}, want: CameraAvailable},
		{name: "liveview local", lan: true, rep: IPCamReport{Present: true, LiveviewLocal: "local"}, want: CameraNotSupported},
		{name: "other liveview", lan: true, rep: IPCamReport{Present: true, LiveviewLocal: "rtsp"}, want: CameraNotSupported},
		{name: "empty ipcam object", lan: true, rep: IPCamReport{Present: true}, want: CameraUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeriveCamera(tt.lan, tt.rep); got != tt.want {
				t.Fatalf("DeriveCamera() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestAllowedCommands_table(t *testing.T) {
	base := func(view ViewState, gcode string, status PrinterStatus, nozzle float64) *PrinterState {
		s := devState(status, gcode)
		s.View = view
		s.NozzleTemp = nozzle
		return s
	}
	tests := []struct {
		name  string
		state *PrinterState
		want  []CommandKind
	}{
		{name: "offline", state: base(ViewOffline, "", PrinterStatusOffline, 0), want: []CommandKind{}},
		{name: "read only", state: base(ViewReadOnly, "RUNNING", PrinterStatusPrinting, 200), want: []CommandKind{}},
		{
			name:  "idle hot nozzle",
			state: base(ViewIdle, "IDLE", PrinterStatusIdle, 180),
			want: []CommandKind{
				CmdSetLight, CmdSetTemp, CmdSetFan, CmdJog, CmdHome, CmdExtrude,
				CmdAMSLoad, CmdAMSUnload, CmdAMSRFID, CmdCalibrate, CmdStartPrint,
			},
		},
		{
			name:  "idle cold nozzle",
			state: base(ViewIdle, "IDLE", PrinterStatusIdle, 169),
			want: []CommandKind{
				CmdSetLight, CmdSetTemp, CmdSetFan, CmdJog, CmdHome,
				CmdAMSLoad, CmdAMSUnload, CmdAMSRFID, CmdCalibrate, CmdStartPrint,
			},
		},
		{
			name:  "finished hot",
			state: base(ViewFinished, "FINISH", PrinterStatusIdle, 170),
			want: []CommandKind{
				CmdSetLight, CmdSetTemp, CmdSetFan, CmdJog, CmdHome, CmdExtrude,
				CmdAMSLoad, CmdAMSUnload, CmdAMSRFID,
			},
		},
		{
			name:  "printing",
			state: base(ViewPrinting, "RUNNING", PrinterStatusPrinting, 220),
			want: []CommandKind{
				CmdPause, CmdStop, CmdSetSpeed, CmdSetLight, CmdSetTemp, CmdSetFan,
				CmdAMSRFID, CmdSkipObjects,
			},
		},
		{
			name:  "paused",
			state: base(ViewPaused, "PAUSE", PrinterStatusPaused, 220),
			want: []CommandKind{
				CmdResume, CmdStop, CmdSetSpeed, CmdSetLight, CmdSetTemp, CmdSetFan,
				CmdAMSLoad, CmdAMSUnload, CmdAMSRFID, CmdSkipObjects,
			},
		},
		{
			name:  "error while paused",
			state: base(ViewError, "PAUSE", PrinterStatusPaused, 220),
			want: []CommandKind{
				CmdResume, CmdStop, CmdSetLight, CmdSetTemp, CmdSetFan, CmdAMSRFID, CmdClearError,
			},
		},
		{
			name:  "error while printing",
			state: base(ViewError, "RUNNING", PrinterStatusError, 220),
			want: []CommandKind{
				CmdStop, CmdSetLight, CmdSetTemp, CmdSetFan, CmdAMSRFID, CmdClearError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AllowedCommands(tt.state)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("allowed = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestAllowedCommands_protocolFilter(t *testing.T) {
	s := devState(PrinterStatusPrinting, "RUNNING")
	s.View = ViewPrinting
	s.Caps.Supported = []CommandKind{CmdPause, CmdResume, CmdStop}
	got := AllowedCommands(s)
	want := []CommandKind{CmdPause, CmdStop}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("allowed = %#v, want %#v", got, want)
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    Command
		wantErr string
	}{
		{name: "pause", body: `{"kind":"pause"}`, want: Pause{}},
		{name: "resume", body: `{"kind":"resume"}`, want: Resume{}},
		{name: "stop", body: `{"kind":"stop"}`, want: Stop{}},
		{name: "speed silent", body: `{"kind":"set_speed","level":1}`, want: SetSpeed{Level: 1}},
		{name: "speed ludicrous", body: `{"kind":"set_speed","level":4}`, want: SetSpeed{Level: 4}},
		{name: "light on", body: `{"kind":"set_light","node":"chamber_light","on":true}`, want: SetLight{Node: LightChamber, On: true}},
		{name: "light off", body: `{"kind":"set_light","node":"work_light","on":false}`, want: SetLight{Node: LightWork, On: false}},
		{name: "second chamber light", body: `{"kind":"set_light","node":"chamber_light2","on":true}`, want: SetLight{Node: LightChamber2, On: true}},
		{name: "empty", body: ``, wantErr: "command body is empty"},
		{name: "level zero", body: `{"kind":"set_speed","level":0}`, wantErr: "speed level must be from 1 to 4"},
		{name: "level five", body: `{"kind":"set_speed","level":5}`, wantErr: "speed level must be from 1 to 4"},
		{name: "level missing", body: `{"kind":"set_speed"}`, wantErr: "set_speed requires level"},
		{name: "light missing on", body: `{"kind":"set_light","node":"chamber_light"}`, wantErr: "set_light requires on"},
		{name: "bad node", body: `{"kind":"set_light","node":"heatbed_light","on":true}`, wantErr: "light node must be chamber_light, chamber_light2, or work_light"},
		{name: "gcode kind", body: `{"kind":"gcode"}`, wantErr: "raw gcode is not accepted"},
		{name: "gcode line kind", body: `{"kind":"gcode_line"}`, wantErr: "raw gcode is not accepted"},
		{name: "gcode field", body: `{"kind":"pause","gcode":"G28"}`, wantErr: "invalid command"},
		{name: "unknown kind", body: `{"kind":"set_temp"}`, wantErr: `unknown command "set_temp"`},
		{name: "pause with level", body: `{"kind":"pause","level":2}`, wantErr: "pause, resume, and stop take no extra fields"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCommand([]byte(tt.body))
			if tt.wantErr != "" {
				var pe *CommandParseError
				if !errors.As(err, &pe) || pe.Error() != tt.wantErr {
					t.Fatalf("err = %v, want %s", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCommandAllowed(t *testing.T) {
	s := devState(PrinterStatusPrinting, "RUNNING")
	s.View = ViewPrinting
	if !CommandAllowed(s, Pause{}) {
		t.Fatal("pause should be allowed while printing")
	}
	if CommandAllowed(s, Resume{}) {
		t.Fatal("resume should be refused while printing")
	}
	if CommandAllowed(nil, Pause{}) {
		t.Fatal("nil state")
	}
}
