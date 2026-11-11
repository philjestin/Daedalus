package printer

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cadimartech/rankos/internal/model"
	"github.com/google/uuid"
)

// Client defines the interface for printer communication.
type Client interface {
	// Connect establishes connection to the printer.
	Connect() error
	// Disconnect closes the connection.
	Disconnect() error
	// GetStatus retrieves current printer status.
	GetStatus() (*model.PrinterState, error)
	// StartJob sends a file to print.
	StartJob(filename string, filepath string) error
	// PauseJob pauses the current print.
	PauseJob() error
	// ResumeJob resumes a paused print.
	ResumeJob() error
	// CancelJob cancels the current print.
	CancelJob() error
	// Send runs one closed command. Pause, resume, and stop apply to any
	// print on the printer, not only a job RankOS created.
	Send(model.Command) error
	// SetStatusCallback sets callback for status updates.
	SetStatusCallback(func(*model.PrinterState))
}

// ErrNotConnected means there is no live session for the printer.
var ErrNotConnected = errors.New("printer not connected")

// StatusChangeCallback is called when a printer's status changes.
// Receives the new state and the previous state (nil if first update).
type StatusChangeCallback func(newState *model.PrinterState, oldState *model.PrinterState)

// Manager manages connections to multiple printers.
type Manager struct {
	mu              sync.RWMutex
	clients         map[uuid.UUID]Client
	states          map[uuid.UUID]*model.PrinterState
	broadcaster     model.Broadcaster
	statusCallbacks []StatusChangeCallback
}

// NewManager creates a new printer manager.
func NewManager() *Manager {
	return &Manager{
		clients:         make(map[uuid.UUID]Client),
		states:          make(map[uuid.UUID]*model.PrinterState),
		statusCallbacks: []StatusChangeCallback{},
	}
}

// OnStatusChange registers a callback for printer status changes.
// This allows services to react to printer state transitions (e.g., detect failures).
func (m *Manager) OnStatusChange(cb StatusChangeCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusCallbacks = append(m.statusCallbacks, cb)
}

// SetBroadcaster sets the broadcaster for real-time updates.
func (m *Manager) SetBroadcaster(b model.Broadcaster) {
	m.broadcaster = b
}

// broadcast sends an event to all connected WebSocket clients.
func (m *Manager) broadcast(eventType string, data interface{}) {
	if m.broadcaster != nil {
		m.broadcaster.Broadcast(model.BroadcastEvent{
			Type: eventType,
			Data: data,
		})
	}
}

// Connect establishes connection to a printer.
// An existing client for the same printer is disconnected first so a settings
// change replaces the MQTT session instead of leaking it.
// The manager lock is not held across the network handshake; status callbacks
// need it as soon as the first report arrives.
func (m *Manager) Connect(p *model.Printer) error {
	m.mu.Lock()
	old, hasOld := m.clients[p.ID]
	if hasOld {
		delete(m.clients, p.ID)
	}
	m.mu.Unlock()
	if hasOld {
		old.Disconnect() //nolint:errcheck // replacing the session
	}

	client, err := newClient(p)
	if err != nil {
		return err
	}
	if client == nil {
		m.mu.Lock()
		m.states[p.ID] = disconnectedState(p)
		m.mu.Unlock()
		return nil
	}

	// Ignore events from a client that has since been replaced. Disconnect
	// fires connection-lost asynchronously, and that must not mark the new
	// session offline.
	client.SetStatusCallback(func(state *model.PrinterState) {
		m.mu.Lock()
		if current, ok := m.clients[p.ID]; !ok || current != client {
			m.mu.Unlock()
			return
		}
		oldState := m.states[p.ID]
		m.states[p.ID] = state
		callbacks := make([]StatusChangeCallback, len(m.statusCallbacks))
		copy(callbacks, m.statusCallbacks)
		m.mu.Unlock()

		slog.Info("printer status update", "printer_id", p.ID, "status", state.Status, "progress", state.Progress)

		// Broadcast state change to WebSocket clients
		m.broadcast(model.EventPrinterStateUpdated, state)

		// Notify registered status change listeners
		for _, cb := range callbacks {
			cb(state, oldState)
		}
	})

	// Connect
	if err := client.Connect(); err != nil {
		slog.Error("failed to connect to printer", "printer_id", p.ID, "error", err)
		m.mu.Lock()
		m.states[p.ID] = disconnectedState(p)
		m.mu.Unlock()
		return err
	}

	// Publish the client before reading status so a report that arrives during
	// the handshake is applied, and so GetStatus does not run while m.mu is held
	// (the status callback takes m.mu and must not deadlock with the client lock).
	m.mu.Lock()
	m.clients[p.ID] = client
	m.mu.Unlock()

	state, stateErr := client.GetStatus()
	m.mu.Lock()
	if stateErr == nil {
		if _, exists := m.states[p.ID]; !exists {
			m.states[p.ID] = state
		} else {
			state = m.states[p.ID]
		}
	}
	m.mu.Unlock()

	if stateErr == nil {
		m.broadcast(model.EventPrinterStateUpdated, state)
	}

	slog.Info("connected to printer", "printer_id", p.ID, "type", p.ConnectionType)

	// Broadcast printer connected event
	m.broadcast(model.EventPrinterConnected, map[string]interface{}{
		"printer_id": p.ID,
	})

	return nil
}

// newClient builds the protocol client for a printer.
// A nil client with a nil error means the printer is manual and has nothing to dial.
func newClient(p *model.Printer) (Client, error) {
	switch p.ConnectionType {
	case model.ConnectionTypeOctoPrint:
		return NewOctoPrintClient(p.ID, p.ConnectionURI, p.APIKey), nil
	case model.ConnectionTypeBambuLAN:
		client := NewBambuClient(p.ID, p.ConnectionURI, p.APIKey, p.SerialNumber)
		client.SetModel(p.Model)
		return client, nil
	case model.ConnectionTypeBambuCloud:
		// For cloud printers: ConnectionURI = MQTT username, APIKey = auth token
		client := NewBambuCloudClient(p.ID, p.SerialNumber, p.ConnectionURI, p.APIKey)
		client.SetModel(p.Model)
		return client, nil
	case model.ConnectionTypeMoonraker:
		return NewMoonrakerClient(p.ID, p.ConnectionURI), nil
	case model.ConnectionTypeManual:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported connection type: %s", p.ConnectionType)
	}
}

// Disconnect closes connection to a printer.
func (m *Manager) Disconnect(id uuid.UUID) {
	m.mu.Lock()
	client, ok := m.clients[id]
	if ok {
		delete(m.clients, id)
	}
	delete(m.states, id)
	m.mu.Unlock()

	if ok {
		client.Disconnect() //nolint:errcheck // best-effort cleanup
	}

	// Broadcast printer disconnected event
	m.broadcast(model.EventPrinterDisconnected, map[string]interface{}{
		"printer_id": id,
	})
}

// DisconnectAll closes all printer connections. Used during graceful shutdown.
func (m *Manager) DisconnectAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, client := range m.clients {
		slog.Info("disconnecting printer", "printer_id", id)
		client.Disconnect() //nolint:errcheck // best-effort shutdown cleanup
		delete(m.clients, id)
	}
	// Clear all states
	for id := range m.states {
		delete(m.states, id)
	}
	slog.Info("all printers disconnected")
}

// GetState retrieves current state for a printer.
func (m *Manager) GetState(id uuid.UUID) (*model.PrinterState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if state, ok := m.states[id]; ok {
		return state, nil
	}
	return nil, fmt.Errorf("printer not found")
}

// GetAllStates retrieves current state for all printers.
func (m *Manager) GetAllStates() map[uuid.UUID]*model.PrinterState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[uuid.UUID]*model.PrinterState)
	for id, state := range m.states {
		result[id] = state
	}
	return result
}

// StartJob sends a print job to a printer.
func (m *Manager) StartJob(printerID uuid.UUID, filename string, filepath string) error {
	m.mu.RLock()
	client, ok := m.clients[printerID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("printer not connected")
	}

	return client.StartJob(filename, filepath)
}

// Send runs a command after checking it against the current view.
// The check runs on every call, including pause, resume, and stop from the job queue.
func (m *Manager) Send(printerID uuid.UUID, cmd model.Command) error {
	m.mu.RLock()
	client, ok := m.clients[printerID]
	state := m.states[printerID]
	m.mu.RUnlock()
	if !ok || client == nil {
		return ErrNotConnected
	}
	if state == nil {
		return ErrNotConnected
	}
	fresh := *state
	model.RefreshView(&fresh, time.Now())
	if !model.CommandAllowed(&fresh, cmd) {
		return model.ErrCommandNotAllowed
	}
	slog.Info("printer command",
		"printer_id", printerID,
		"kind", model.CommandKindOf(cmd),
	)
	return client.Send(cmd)
}

// PauseJob pauses the current print on a printer.
func (m *Manager) PauseJob(printerID uuid.UUID) error {
	return m.Send(printerID, model.Pause{})
}

// ResumeJob resumes a paused print on a printer.
func (m *Manager) ResumeJob(printerID uuid.UUID) error {
	return m.Send(printerID, model.Resume{})
}

// CancelJob cancels the current print on a printer.
func (m *Manager) CancelJob(printerID uuid.UUID) error {
	return m.Send(printerID, model.Stop{})
}
