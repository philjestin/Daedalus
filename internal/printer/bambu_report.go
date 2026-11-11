package printer

import (
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/cadimartech/rankos/internal/model"
	"github.com/google/uuid"
)

// mqttSignatureBit is print.fun bit 29 (0x20000000). Set means the firmware
// requires a signed command, which is Developer Mode off.
// H2D reports fun "3EC1AFFF9CFF" with the bit set and "3EC18FFF9CFF" with it clear.
const mqttSignatureBit = uint64(0x20000000)

// flag3AuthEnabled is print.flag3 bit 4. Some firmware leaves fun empty and
// uses this bit for the same "signature required" signal.
// Bit 16 (0x10000) only means the firmware supports that system. It stays set
// with Developer Mode on, so it must not force read-only.
const flag3AuthEnabled = 0x10

func initialBambuState(id uuid.UUID, cloud bool) *model.PrinterState {
	cam := model.CameraUnknown
	if cloud {
		cam = model.CameraNotSupported
	}
	return &model.PrinterState{
		PrinterID: id,
		Status:    model.PrinterStatusOffline,
		UpdatedAt: time.Now(),
		View:      model.ViewOffline,
		Caps:      model.Capabilities{DevMode: true, Nozzles: 1},
		Allowed:   []model.CommandKind{},
		Camera:    model.CameraInfo{Availability: cam},
	}
}

// familyCapabilities is the static table for a model name.
// The report can add a second light, a second nozzle, or fans the name did not imply.
func familyCapabilities(modelName string) model.Capabilities {
	u := strings.ToUpper(modelName)
	u = strings.ReplaceAll(u, " ", "")
	u = strings.ReplaceAll(u, "-", "")
	u = strings.ReplaceAll(u, "_", "")
	caps := model.Capabilities{DevMode: true, Nozzles: 1}
	switch {
	case strings.Contains(u, "H2D"), strings.Contains(u, "H2C"), strings.Contains(u, "X2D"):
		caps.Nozzles = 2
		caps.ChamberHeater = true
		caps.SecondLight = true
		caps.AuxFan = true
		caps.ChamberFan = true
		caps.AirductModes = true
	case strings.Contains(u, "H2"), strings.Contains(u, "H3"):
		caps.ChamberHeater = true
		caps.SecondLight = strings.Contains(u, "H2")
		caps.AuxFan = true
		caps.ChamberFan = true
		caps.AirductModes = true
	case strings.Contains(u, "P2"):
		caps.AuxFan = true
		caps.ChamberFan = true
		caps.AirductModes = true
	case strings.Contains(u, "X1E"):
		caps.ChamberHeater = true
		caps.AuxFan = true
		caps.ChamberFan = true
	case strings.Contains(u, "X1"), strings.Contains(u, "P1S"):
		caps.AuxFan = true
		caps.ChamberFan = true
	case strings.Contains(u, "P1P"):
		caps.AuxFan = true
	}
	return caps
}

// SetModel selects the capability table for this printer.
func (c *BambuClient) SetModel(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.modelName = name
	c.baseCaps = familyCapabilities(name)
}

func (c *BambuClient) effectiveCaps() model.Capabilities {
	caps := c.baseCaps
	if caps.Nozzles < 1 {
		caps.Nozzles = 1
	}
	if c.nozzlesFromReport > caps.Nozzles {
		caps.Nozzles = c.nozzlesFromReport
	}
	if c.secondLight {
		caps.SecondLight = true
	}
	if c.sawAux {
		caps.AuxFan = true
	}
	if c.sawChamberFan {
		caps.ChamberFan = true
	}
	caps.DevMode = !c.signatureRequired
	return caps
}

// absorb folds security, camera, and device-block temperatures into state.
// Caller holds c.mu. Partial reports keep the last camera and developer-mode bit.
func (c *BambuClient) absorb(state *model.PrinterState, print *BambuPrintStatus) {
	if print != nil {
		c.noteSecurity(print)
		c.noteHardware(state, print)
		c.noteCamera(print)
		c.applyDevice(state, print)
	}
	state.Caps = c.effectiveCaps()
	switch {
	case c.cloudMode:
		state.Camera = model.CameraInfo{Availability: model.CameraNotSupported}
	case c.cameraSeen:
		state.Camera = c.camera
	case state.Camera.Availability == "":
		state.Camera = model.CameraInfo{Availability: model.CameraUnknown}
	}
	if len(state.Lights) > 0 {
		state.ChamberLight = lightMode(state.Lights, model.LightChamber, model.LightChamber2)
		state.WorkLight = lightMode(state.Lights, model.LightWork)
	}
}

func (c *BambuClient) noteSecurity(print *BambuPrintStatus) {
	// A non-empty fun value is authoritative. An empty or missing fun falls
	// through to flag3, which some firmware sends instead.
	if print.present != nil && print.has("fun") && print.Fun != "" {
		v, ok := parseHex(print.Fun)
		if !ok {
			return
		}
		c.signatureRequired = v&mqttSignatureBit != 0
		return
	}
	if print.present != nil && print.has("flag3") {
		c.signatureRequired = print.Flag3&flag3AuthEnabled != 0
	}
}

func parseHex(s string) (uint64, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	if s == "" {
		return 0, false
	}
	v := uint64(0)
	for _, r := range s {
		v <<= 4
		switch {
		case r >= '0' && r <= '9':
			v |= uint64(r - '0')
		case r >= 'a' && r <= 'f':
			v |= uint64(r-'a') + 10
		case r >= 'A' && r <= 'F':
			v |= uint64(r-'A') + 10
		default:
			return 0, false
		}
	}
	return v, true
}

func (c *BambuClient) noteHardware(state *model.PrinterState, print *BambuPrintStatus) {
	if print.present != nil && print.has("big_fan1_speed") {
		c.sawAux = true
	}
	if print.present != nil && print.has("big_fan2_speed") {
		c.sawChamberFan = true
	}
	for _, l := range state.Lights {
		if l.Node == model.LightChamber2 {
			c.secondLight = true
		}
	}
}

func (c *BambuClient) noteCamera(print *BambuPrintStatus) {
	if c.cloudMode {
		c.camera = model.CameraInfo{Availability: model.CameraNotSupported}
		c.cameraSeen = true
		return
	}
	if print.present != nil && !print.has("ipcam") {
		return
	}
	if print.present == nil && print.IPCam == nil {
		return
	}
	rep := model.IPCamReport{Present: true}
	if print.IPCam != nil {
		rep.RTSPURL = print.IPCam.RTSPURL
		if print.IPCam.Liveview != nil {
			rep.LiveviewLocal = print.IPCam.Liveview.Local
		}
	}
	avail := model.DeriveCamera(true, rep)
	c.noteUnrecognizedCamera(rep, avail)
	c.camera = model.CameraInfo{Availability: avail}
	c.cameraSeen = true
}

func (c *BambuClient) noteUnrecognizedCamera(rep model.IPCamReport, avail model.CameraAvailability) {
	if avail != model.CameraNotSupported {
		return
	}
	note := ""
	if rtsp := strings.TrimSpace(rep.RTSPURL); rtsp != "" && !strings.EqualFold(rtsp, "disable") && !strings.HasPrefix(strings.ToLower(rtsp), "rtsps://") {
		note = "rtsp_url"
	} else if local := strings.ToLower(strings.TrimSpace(rep.LiveviewLocal)); local != "" && local != "local" && local != "none" && local != "disabled" && local != "rtsps" {
		note = local
	}
	if note == "" || note == c.lastCameraNote {
		return
	}
	c.lastCameraNote = note
	slog.Info("unrecognized ipcam value", "printer_id", c.printerID, "field", note)
}

func (c *BambuClient) applyDevice(state *model.PrinterState, print *BambuPrintStatus) {
	raw := print.Device
	if len(raw) == 0 || string(raw) == "null" {
		return
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return
	}
	if bed, ok := root["bed"]; ok {
		if cur, tgt, ok := packedTempField(bed); ok {
			state.BedTemp = cur
			state.BedTargetTemp = tgt
		}
	}
	if ctc, ok := root["ctc"]; ok {
		if cur, tgt, ok := packedTempField(ctc); ok {
			state.ChamberTemp = cur
			state.ChamberTargetTemp = tgt
		}
	}
	ex, ok := root["extruder"]
	if !ok {
		return
	}
	var block struct {
		Info []struct {
			ID   int  `json:"id"`
			Temp *int `json:"temp"`
		} `json:"info"`
		State int `json:"state"`
	}
	if err := json.Unmarshal(ex, &block); err != nil {
		return
	}
	count := block.State & 0x0F
	if count == 0 {
		count = len(block.Info)
	}
	if count > c.nozzlesFromReport {
		c.nozzlesFromReport = count
	}
	active := (block.State >> 4) & 0x0F
	chosen := -1
	for i, entry := range block.Info {
		if entry.Temp == nil {
			continue
		}
		if entry.ID == active {
			chosen = i
			break
		}
		if chosen < 0 && entry.ID == 0 {
			chosen = i
		}
	}
	if chosen < 0 {
		return
	}
	cur, tgt := splitPacked(*block.Info[chosen].Temp)
	state.NozzleTemp = cur
	state.NozzleTargetTemp = tgt
}

func packedTempField(raw json.RawMessage) (float64, float64, bool) {
	var wrap struct {
		Info struct {
			Temp *int `json:"temp"`
		} `json:"info"`
		Temp *int `json:"temp"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return 0, 0, false
	}
	v := wrap.Info.Temp
	if v == nil {
		v = wrap.Temp
	}
	if v == nil {
		return 0, 0, false
	}
	cur, tgt := splitPacked(*v)
	return cur, tgt, true
}

// splitPacked separates a Bambu packed temperature. The low 16 bits are the
// current reading and the high 16 bits are the target.
func splitPacked(v int) (current, target float64) {
	if v < 0 {
		return 0, 0
	}
	return float64(v & 0xFFFF), float64((v >> 16) & 0xFFFF)
}

func lightMode(lights []model.LightState, nodes ...string) string {
	found := make(map[string]string, len(lights))
	for _, l := range lights {
		found[l.Node] = l.Mode
	}
	var modes []string
	for _, n := range nodes {
		if m := found[n]; m != "" {
			modes = append(modes, m)
		}
	}
	if len(modes) == 0 {
		return ""
	}
	for _, m := range modes {
		if m == "on" || m == "flashing" {
			return m
		}
	}
	return modes[0]
}

// snapshotLocked copies the last report and marks it offline once it is stale.
// The stored report is left intact so the next delta can still merge.
// Caller holds c.mu.
func (c *BambuClient) snapshotLocked(now time.Time) *model.PrinterState {
	if c.lastState == nil {
		st := initialBambuState(c.printerID, c.cloudMode)
		st.Caps = c.effectiveCaps()
		st.UpdatedAt = now
		model.RefreshView(st, now)
		return st
	}
	cp := *c.lastState
	after := c.staleAfter
	if after <= 0 {
		after = model.ReportStaleAfter
	}
	if cp.Status != model.PrinterStatusOffline && !cp.UpdatedAt.IsZero() && now.Sub(cp.UpdatedAt) > after {
		cp.Status = model.PrinterStatusOffline
	}
	model.RefreshView(&cp, now)
	return &cp
}

func (c *BambuClient) ensureStaleWatch() {
	c.mu.Lock()
	if c.staleWatchStarted {
		c.mu.Unlock()
		return
	}
	c.staleWatchStarted = true
	interval := c.staleInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	stop := c.stopChan
	c.mu.Unlock()
	if stop == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				c.notifyIfStale()
			}
		}
	}()
}

func (c *BambuClient) notifyIfStale() {
	c.mu.Lock()
	after := c.staleAfter
	if after <= 0 {
		after = model.ReportStaleAfter
	}
	silent := c.lastState != nil &&
		!c.staleNotified &&
		c.lastState.Status != model.PrinterStatusOffline &&
		!c.lastState.UpdatedAt.IsZero() &&
		time.Since(c.lastState.UpdatedAt) > after
	if !silent {
		c.mu.Unlock()
		return
	}
	c.staleNotified = true
	snap := c.snapshotLocked(time.Now())
	cb := c.statusCallback
	c.mu.Unlock()
	if cb != nil {
		cb(snap)
	}
}

// disconnectedState is the state published when there is no live session.
func disconnectedState(p *model.Printer) *model.PrinterState {
	st := &model.PrinterState{
		PrinterID: p.ID,
		Status:    model.PrinterStatusOffline,
		UpdatedAt: time.Now(),
		Caps:      model.Capabilities{DevMode: true, Nozzles: 1},
		Allowed:   []model.CommandKind{},
	}
	if p.ConnectionType == model.ConnectionTypeBambuLAN || p.ConnectionType == model.ConnectionTypeBambuCloud {
		st.Caps = familyCapabilities(p.Model)
		st.Caps.DevMode = true
		if p.ConnectionType == model.ConnectionTypeBambuCloud {
			st.Camera.Availability = model.CameraNotSupported
		} else {
			st.Camera.Availability = model.CameraUnknown
		}
	} else {
		st.Camera.Availability = model.CameraNotSupported
		if p.ConnectionType == model.ConnectionTypeOctoPrint || p.ConnectionType == model.ConnectionTypeMoonraker {
			st.Caps.Supported = append([]model.CommandKind(nil), model.CmdPause, model.CmdResume, model.CmdStop)
		}
	}
	model.RefreshView(st, time.Now())
	return st
}

// annotateOtherProtocol fills view, camera, and the pause/resume/stop allow-list
// for OctoPrint and Moonraker.
func annotateOtherProtocol(state *model.PrinterState) {
	if state == nil {
		return
	}
	if state.Caps.Nozzles < 1 {
		state.Caps.Nozzles = 1
	}
	state.Caps.DevMode = true
	state.Caps.Supported = append([]model.CommandKind(nil), model.CmdPause, model.CmdResume, model.CmdStop)
	state.Camera = model.CameraInfo{Availability: model.CameraNotSupported}
	model.RefreshView(state, time.Now())
}
