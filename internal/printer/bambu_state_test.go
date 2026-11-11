package printer

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/cadimartech/rankos/internal/model"
	"github.com/google/uuid"
)

func TestFamilyCapabilities(t *testing.T) {
	tests := []struct {
		model string
		heat  bool
		noz   int
		light bool
		air   bool
	}{
		{model: "Bambu Lab P2S", heat: false, noz: 1, light: false, air: true},
		{model: "H2S", heat: true, noz: 1, light: true, air: true},
		{model: "H2D", heat: true, noz: 2, light: true, air: true},
		{model: "H2C", heat: true, noz: 2, light: true, air: true},
		{model: "H3C", heat: true, noz: 1, light: false, air: true},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			caps := familyCapabilities(tt.model)
			if caps.ChamberHeater != tt.heat || caps.Nozzles != tt.noz || caps.SecondLight != tt.light || !caps.AirductModes || caps.AirductModes != tt.air {
				t.Fatalf("caps = %+v", caps)
			}
			if !caps.AuxFan || !caps.ChamberFan {
				t.Fatalf("fans = %+v", caps)
			}
			if caps.MQTTBedTemp {
				t.Fatal("mqtt bed temp should stay false until a report announces it")
			}
		})
	}
}

func TestMerge_P2SIdleThenPrintingDelta(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.0.2.20", "code", "22E8AA000000001")
	client.SetModel("P2S")
	client.ingestReport("device/x/report", []byte(`{
		"print": {
			"gcode_state": "IDLE",
			"stg_cur": -1,
			"fun": "3EC18FFF9CFF",
			"flag3": 221839,
			"mc_percent": 0,
			"mc_remaining_time": 0,
			"bed_temper": 25.0,
			"bed_target_temper": 0,
			"nozzle_temper": 28.0,
			"nozzle_target_temper": 0,
			"chamber_temper": 24.0,
			"layer_num": 0,
			"total_layer_num": 0,
			"cooling_fan_speed": "0",
			"big_fan1_speed": "0",
			"big_fan2_speed": "0",
			"spd_lvl": 2,
			"print_error": 0,
			"hms": [],
			"ipcam": {
				"rtsp_url": "rtsps://printer/streaming/live/1",
				"liveview": {"local": "rtsps"}
			},
			"lights_report": [{"node": "chamber_light", "mode": "off"}],
			"ams": {
				"tray_now": "0",
				"ams": [{
					"id": "0",
					"humidity": "18",
					"temp": "27",
					"tray": [{
						"id": "0",
						"tray_type": "PLA",
						"tray_color": "FFFFFFFF",
						"tray_sub_brands": "Bambu",
						"remain": 82
					}]
				}]
			}
		}
	}`))

	state, err := client.GetStatus()
	if err != nil {
		t.Fatal(err)
	}
	if state.View != model.ViewIdle {
		t.Fatalf("view = %s", state.View)
	}
	if !state.Caps.DevMode || state.Caps.ChamberHeater || state.Caps.Nozzles != 1 || state.Caps.SecondLight {
		t.Fatalf("caps = %+v", state.Caps)
	}
	if state.Camera.Availability != model.CameraAvailable {
		t.Fatalf("camera = %s", state.Camera.Availability)
	}
	if state.BedTemp != 25 || state.NozzleTemp != 28 || state.ChamberTemp != 24 {
		t.Fatalf("temps bed=%v nozzle=%v chamber=%v", state.BedTemp, state.NozzleTemp, state.ChamberTemp)
	}
	if state.ChamberLight != "off" {
		t.Fatalf("light = %s", state.ChamberLight)
	}
	if state.Stage != "Idle" {
		t.Fatalf("stage = %s", state.Stage)
	}
	if state.AMS == nil || len(state.AMS.Units) != 1 || len(state.AMS.Units[0].Trays) != 1 {
		t.Fatalf("ams = %+v", state.AMS)
	}
	tray := state.AMS.Units[0].Trays[0]
	if tray.MaterialType != "PLA" || tray.ColorHex != "FFFFFFFF" || tray.Remain != 82 {
		t.Fatalf("tray = %+v", tray)
	}
	if state.AMS.Units[0].Humidity != 18 {
		t.Fatalf("humidity = %d", state.AMS.Units[0].Humidity)
	}

	client.ingestReport("device/x/report", []byte(`{
		"print": {
			"gcode_state": "RUNNING",
			"mc_percent": 46,
			"mc_remaining_time": 112,
			"layer_num": 142,
			"total_layer_num": 310,
			"stg_cur": 2,
			"subtask_name": "keychain_logo.3mf",
			"spd_lvl": 3
		}
	}`))
	state, _ = client.GetStatus()
	if state.View != model.ViewPrinting || state.Status != model.PrinterStatusPrinting {
		t.Fatalf("view %s status %s", state.View, state.Status)
	}
	if state.NozzleTemp != 28 || state.BedTemp != 25 || state.ChamberTemp != 24 {
		t.Fatalf("delta wiped temps: %+v", state)
	}
	if state.Camera.Availability != model.CameraAvailable || !state.Caps.DevMode {
		t.Fatalf("delta wiped camera or dev mode: %+v %+v", state.Camera, state.Caps)
	}
	if state.Progress != 46 || state.LayerNum != 142 || state.TotalLayerNum != 310 {
		t.Fatalf("progress %+v", state)
	}
	if state.TimeLeft != 112*60 {
		t.Fatalf("time left %d", state.TimeLeft)
	}
	if state.FinishAt == nil {
		t.Fatal("missing finish_at")
	}
	if state.Stage != "Heating bed" || state.SpeedLevel != 3 || state.CurrentFile != "keychain_logo.3mf" {
		t.Fatalf("stage %s speed %d file %s", state.Stage, state.SpeedLevel, state.CurrentFile)
	}
	if state.ChamberLight != "off" {
		t.Fatalf("light cleared: %s", state.ChamberLight)
	}
}

func TestMerge_H2SAndH2D(t *testing.T) {
	h2s := NewBambuClient(uuid.New(), "192.0.2.21", "code", "22E8AA000000002")
	h2s.SetModel("H2S")
	h2s.ingestReport("t", []byte(`{
		"print": {
			"gcode_state": "RUNNING",
			"fun": "3EC18FFF9CFF",
			"stg_cur": 49,
			"chamber_temper": 38,
			"nozzle_temper": 220,
			"nozzle_target_temper": 220,
			"bed_temper": 65,
			"bed_target_temper": 65,
			"ipcam": {"liveview": {"local": "rtsps"}}
		}
	}`))
	state, _ := h2s.GetStatus()
	if !state.Caps.ChamberHeater || state.Caps.Nozzles != 1 || !state.Caps.SecondLight {
		t.Fatalf("h2s caps %+v", state.Caps)
	}
	if state.Stage != "Heating chamber" || state.ChamberTemp != 38 || state.View != model.ViewPrinting {
		t.Fatalf("h2s state stage %s chamber %v view %s", state.Stage, state.ChamberTemp, state.View)
	}
	if state.Camera.Availability != model.CameraAvailable {
		t.Fatalf("h2s camera %s", state.Camera.Availability)
	}

	h2d := NewBambuClient(uuid.New(), "192.0.2.22", "code", "22E8AA000000003")
	h2d.SetModel("H2D")
	// 220/220 nozzle, 65 target / 60 current bed, 40 target / 38 current chamber.
	h2d.ingestReport("t", []byte(`{
		"print": {
			"gcode_state": "PAUSE",
			"fun": "3EC18FFF9CFF",
			"stg_cur": 16,
			"mc_percent": 46,
			"lights_report": [
				{"node": "chamber_light", "mode": "off"},
				{"node": "chamber_light2", "mode": "on"},
				{"node": "work_light", "mode": "off"}
			],
			"device": {
				"bed": {"info": {"temp": 4259900}},
				"ctc": {"info": {"temp": 2621478}},
				"extruder": {
					"state": 2,
					"info": [
						{"id": 0, "temp": 14418140},
						{"id": 1, "temp": 6553600}
					]
				}
			}
		}
	}`))
	state, _ = h2d.GetStatus()
	if state.View != model.ViewPaused {
		t.Fatalf("view %s", state.View)
	}
	if state.Caps.Nozzles != 2 || !state.Caps.SecondLight || !state.Caps.ChamberHeater {
		t.Fatalf("h2d caps %+v", state.Caps)
	}
	if state.NozzleTemp != 220 || state.NozzleTargetTemp != 220 {
		t.Fatalf("nozzle %v/%v", state.NozzleTemp, state.NozzleTargetTemp)
	}
	if state.BedTemp != 60 || state.BedTargetTemp != 65 {
		t.Fatalf("bed %v/%v", state.BedTemp, state.BedTargetTemp)
	}
	if state.ChamberTemp != 38 || state.ChamberTargetTemp != 40 {
		t.Fatalf("chamber %v/%v", state.ChamberTemp, state.ChamberTargetTemp)
	}
	if state.ChamberLight != "on" || state.WorkLight != "off" {
		t.Fatalf("lights chamber %s work %s", state.ChamberLight, state.WorkLight)
	}
	if state.Stage != "Paused by user" {
		t.Fatalf("stage %s", state.Stage)
	}
}

func TestMerge_readOnlyAndFlag3(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.0.2.20", "code", "22E8AA000000001")
	client.SetModel("P2S")
	client.ingestReport("t", []byte(`{"print":{"gcode_state":"RUNNING","fun":"3EC1AFFF9CFF"}}`))
	state, _ := client.GetStatus()
	if state.Status != model.PrinterStatusPrinting || state.View != model.ViewReadOnly || state.Caps.DevMode {
		t.Fatalf("status %s view %s dev %v", state.Status, state.View, state.Caps.DevMode)
	}
	if len(state.Allowed) != 0 {
		t.Fatalf("allowed %v", state.Allowed)
	}

	// fun wins over a flag3 bit that would say the opposite.
	client.ingestReport("t", []byte(`{"print":{"gcode_state":"IDLE","fun":"3EC18FFF9CFF","flag3":221855}}`))
	state, _ = client.GetStatus()
	if state.View != model.ViewIdle || !state.Caps.DevMode {
		t.Fatalf("fun should clear read-only, view %s dev %v", state.View, state.Caps.DevMode)
	}

	emptyFun := NewBambuClient(uuid.New(), "192.0.2.20", "code", "22E8AA000000004")
	emptyFun.SetModel("P2S")
	// Bit 16 is set in both samples. Bit 4 is clear here: Developer Mode on.
	emptyFun.ingestReport("t", []byte(`{"print":{"gcode_state":"IDLE","flag3":221839}}`))
	state, _ = emptyFun.GetStatus()
	if state.View != model.ViewIdle || !state.Caps.DevMode {
		t.Fatalf("flag3 bit 16 alone must not force read-only: view %s dev %v", state.View, state.Caps.DevMode)
	}
	emptyFun.ingestReport("t", []byte(`{"print":{"gcode_state":"IDLE","flag3":221855}}`))
	state, _ = emptyFun.GetStatus()
	if state.View != model.ViewReadOnly || state.Caps.DevMode {
		t.Fatalf("flag3 bit 4 should be read-only: view %s dev %v", state.View, state.Caps.DevMode)
	}
}

func TestMerge_cameraAndErrors(t *testing.T) {
	off := NewBambuClient(uuid.New(), "192.0.2.20", "code", "sn")
	off.SetModel("P2S")
	off.ingestReport("t", []byte(`{"print":{"gcode_state":"IDLE","fun":"3EC18FFF9CFF","ipcam":{"rtsp_url":"disable","liveview":{"local":"rtsps"}}}}`))
	state, _ := off.GetStatus()
	if state.Camera.Availability != model.CameraLiveviewOff {
		t.Fatalf("camera %s", state.Camera.Availability)
	}

	local := NewBambuClient(uuid.New(), "192.0.2.20", "code", "sn")
	local.ingestReport("t", []byte(`{"print":{"gcode_state":"IDLE","ipcam":{"liveview":{"local":"local"}}}}`))
	state, _ = local.GetStatus()
	if state.Camera.Availability != model.CameraNotSupported {
		t.Fatalf("a1 camera %s", state.Camera.Availability)
	}

	cloud := NewBambuCloudClient(uuid.New(), "sn", "u_1", "token")
	cloud.SetModel("P2S")
	cloud.ingestReport("t", []byte(`{"print":{"gcode_state":"IDLE","ipcam":{"rtsp_url":"rtsps://printer/streaming/live/1"}}}`))
	state, _ = cloud.GetStatus()
	if state.Camera.Availability != model.CameraNotSupported {
		t.Fatalf("cloud camera %s", state.Camera.Availability)
	}

	unknown := NewBambuClient(uuid.New(), "192.0.2.20", "code", "sn")
	unknown.ingestReport("t", []byte(`{"print":{"gcode_state":"IDLE","bed_temper":20}}`))
	state, _ = unknown.GetStatus()
	if state.Camera.Availability != model.CameraUnknown {
		t.Fatalf("camera before ipcam %s", state.Camera.Availability)
	}

	errp := NewBambuClient(uuid.New(), "192.0.2.20", "code", "sn")
	errp.SetModel("H2S")
	errp.ingestReport("t", []byte(`{
		"print": {
			"gcode_state": "RUNNING",
			"fun": "3EC18FFF9CFF",
			"print_error": 0,
			"hms": [{"attr": 1, "code": 100, "module": 7, "severity": 1}]
		}
	}`))
	state, _ = errp.GetStatus()
	if state.View != model.ViewError || len(state.HMSErrors) != 1 {
		t.Fatalf("view %s hms %v", state.View, state.HMSErrors)
	}
	if !containsKind(state.Allowed, model.CmdClearError) || !containsKind(state.Allowed, model.CmdStop) {
		t.Fatalf("allowed %v", state.Allowed)
	}
	if containsKind(state.Allowed, model.CmdResume) {
		t.Fatal("resume while printing should stay off")
	}

	done := NewBambuClient(uuid.New(), "192.0.2.20", "code", "sn")
	done.SetModel("P2S")
	done.ingestReport("t", []byte(`{"print":{"gcode_state":"FINISH","fun":"3EC18FFF9CFF","mc_percent":100,"print_error":0}}`))
	state, _ = done.GetStatus()
	if state.Status != model.PrinterStatusIdle || state.View != model.ViewFinished {
		t.Fatalf("status %s view %s", state.Status, state.View)
	}
}

func TestSnapshot_staleReportIsOfflineWithoutForgettingTemps(t *testing.T) {
	client := NewBambuClient(uuid.New(), "192.0.2.20", "code", "sn")
	client.ingestReport("t", []byte(`{"print":{"gcode_state":"RUNNING","fun":"3EC18FFF9CFF","nozzle_temper":210}}`))
	client.mu.Lock()
	client.lastState.UpdatedAt = time.Now().Add(-time.Minute)
	client.mu.Unlock()

	state, _ := client.GetStatus()
	if state.Status != model.PrinterStatusOffline || state.View != model.ViewOffline {
		t.Fatalf("status %s view %s", state.Status, state.View)
	}
	client.ingestReport("t", []byte(`{"print":{"mc_percent":50}}`))
	state, _ = client.GetStatus()
	if state.NozzleTemp != 210 || state.View != model.ViewPrinting {
		t.Fatalf("merge after stale: nozzle %v view %s", state.NozzleTemp, state.View)
	}
}

func TestBambuCommandPayloads(t *testing.T) {
	tests := []struct {
		name string
		cmd  model.Command
		want string
	}{
		{name: "pause", cmd: model.Pause{}, want: `{"print":{"sequence_id":"0","command":"pause","param":""}}`},
		{name: "resume", cmd: model.Resume{}, want: `{"print":{"sequence_id":"0","command":"resume","param":""}}`},
		{name: "stop", cmd: model.Stop{}, want: `{"print":{"sequence_id":"0","command":"stop","param":""}}`},
		{name: "speed", cmd: model.SetSpeed{Level: 2}, want: `{"print":{"sequence_id":"0","command":"print_speed","param":"2"}}`},
		{name: "light on", cmd: model.SetLight{Node: model.LightChamber, On: true}, want: `{"system":{"sequence_id":"0","command":"ledctrl","led_node":"chamber_light","led_mode":"on","led_on_time":500,"led_off_time":500,"loop_times":1,"interval_time":1000}}`},
		{name: "light off", cmd: model.SetLight{Node: model.LightWork, On: false}, want: `{"system":{"sequence_id":"0","command":"ledctrl","led_node":"work_light","led_mode":"off","led_on_time":500,"led_off_time":500,"loop_times":1,"interval_time":1000}}`},
		{name: "second light", cmd: model.SetLight{Node: model.LightChamber2, On: true}, want: `{"system":{"sequence_id":"0","command":"ledctrl","led_node":"chamber_light2","led_mode":"on","led_on_time":500,"led_off_time":500,"loop_times":1,"interval_time":1000}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := bambuCommand(tt.cmd, "0")
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tt.want {
				t.Fatalf("payload = %s\nwant    %s", data, tt.want)
			}
		})
	}
	if _, err := bambuCommand(model.SetSpeed{Level: 5}, "0"); err == nil {
		t.Fatal("expected level 5 to fail")
	}
}

func TestManagerSend_rechecksState(t *testing.T) {
	m := NewManager()
	id := uuid.New()
	stub := &stubClient{}
	m.clients[id] = stub
	m.states[id] = &model.PrinterState{
		PrinterID:  id,
		Status:     model.PrinterStatusIdle,
		GcodeState: "IDLE",
		UpdatedAt:  time.Now(),
		Caps:       model.Capabilities{DevMode: true, Nozzles: 1},
	}
	if err := m.Send(id, model.Pause{}); !errors.Is(err, model.ErrCommandNotAllowed) {
		t.Fatalf("idle pause: %v", err)
	}
	if len(stub.sent) != 0 {
		t.Fatalf("sent %d", len(stub.sent))
	}

	m.states[id].Status = model.PrinterStatusPrinting
	m.states[id].GcodeState = "RUNNING"
	if err := m.Send(id, model.Pause{}); err != nil {
		t.Fatal(err)
	}
	if err := m.Send(id, model.SetSpeed{Level: 4}); err != nil {
		t.Fatal(err)
	}
	if err := m.Send(id, model.SetLight{Node: model.LightChamber, On: false}); err != nil {
		t.Fatal(err)
	}
	if len(stub.sent) != 3 {
		t.Fatalf("sent %#v", stub.sent)
	}

	m.states[id].Caps.DevMode = false
	if err := m.Send(id, model.Stop{}); !errors.Is(err, model.ErrCommandNotAllowed) {
		t.Fatalf("read only stop: %v", err)
	}
	if err := m.Send(uuid.New(), model.Pause{}); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("missing printer: %v", err)
	}
}

func containsKind(list []model.CommandKind, kind model.CommandKind) bool {
	for _, k := range list {
		if k == kind {
			return true
		}
	}
	return false
}

type stubClient struct {
	sent []model.Command
}

func (s *stubClient) Connect() error                              { return nil }
func (s *stubClient) Disconnect() error                           { return nil }
func (s *stubClient) GetStatus() (*model.PrinterState, error)     { return nil, nil }
func (s *stubClient) StartJob(string, string) error               { return nil }
func (s *stubClient) PauseJob() error                             { return nil }
func (s *stubClient) ResumeJob() error                            { return nil }
func (s *stubClient) CancelJob() error                            { return nil }
func (s *stubClient) SetStatusCallback(func(*model.PrinterState)) {}
func (s *stubClient) Send(cmd model.Command) error {
	s.sent = append(s.sent, cmd)
	return nil
}
