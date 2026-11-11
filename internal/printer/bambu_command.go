package printer

import (
	"fmt"
	"strconv"
	"time"

	"github.com/cadimartech/rankos/internal/model"
)

// ledBlinkOnMS, ledBlinkOffMS, ledLoopTimes, and ledIntervalMS are the blink
// fields OpenBambuAPI requires on every ledctrl, including a steady on or off.
const (
	ledBlinkOnMS  = 500
	ledBlinkOffMS = 500
	ledLoopTimes  = 1
	ledIntervalMS = 1000
)

// bambuCommand builds the MQTT request for a closed command.
// sequence is the sequence_id string. Tests pass "0" to match OpenBambuAPI.
func bambuCommand(cmd model.Command, sequence string) (BambuCommand, error) {
	switch c := cmd.(type) {
	case model.Pause:
		return printControl("pause", sequence, ""), nil
	case model.Resume:
		return printControl("resume", sequence, ""), nil
	case model.Stop:
		return printControl("stop", sequence, ""), nil
	case model.SetSpeed:
		if c.Level < 1 || c.Level > 4 {
			return BambuCommand{}, &model.CommandParseError{Reason: "speed level must be from 1 to 4"}
		}
		return printControl("print_speed", sequence, strconv.Itoa(c.Level)), nil
	case model.SetLight:
		mode := "off"
		if c.On {
			mode = "on"
		}
		return BambuCommand{
			System: &BambuSystemCmd{
				Sequence:     sequence,
				Command:      "ledctrl",
				LedNode:      c.Node,
				LedMode:      mode,
				LedOnTime:    ledBlinkOnMS,
				LedOffTime:   ledBlinkOffMS,
				LoopTimes:    ledLoopTimes,
				IntervalTime: ledIntervalMS,
			},
		}, nil
	default:
		return BambuCommand{}, model.ErrUnsupported
	}
}

func printControl(command, sequence, param string) BambuCommand {
	p := param
	return BambuCommand{
		Print: &BambuPrintCmd{
			Sequence: sequence,
			Command:  command,
			Param:    &p,
		},
	}
}

// Send publishes one closed command. Pause, resume, and stop apply to whatever
// the printer is doing, including a print RankOS did not start.
func (c *BambuClient) Send(cmd model.Command) error {
	seq := strconv.FormatInt(time.Now().UnixMilli(), 10)
	payload, err := bambuCommand(cmd, seq)
	if err != nil {
		return err
	}
	if err := c.sendCommand(payload); err != nil {
		return fmt.Errorf("send %s: %w", model.CommandKindOf(cmd), err)
	}
	return nil
}
