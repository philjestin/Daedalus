package printer

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
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
	// SetStatusCallback sets callback for status updates.
	SetStatusCallback(func(*model.PrinterState))
}

// Manager manages connections to multiple printers.
type Manager struct {
	mu          sync.RWMutex
	clients     map[uuid.UUID]Client
	states      map[uuid.UUID]*model.PrinterState
	broadcaster model.Broadcaster
}

// NewManager creates a new printer manager.
func NewManager() *Manager {
	return &Manager{
		clients: make(map[uuid.UUID]Client),
		states:  make(map[uuid.UUID]*model.PrinterState),
	}
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
func (m *Manager) Connect(p *model.Printer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create appropriate client based on connection type
	var client Client
	switch p.ConnectionType {
	case model.ConnectionTypeOctoPrint:
		client = NewOctoPrintClient(p.ID, p.ConnectionURI, p.APIKey)
	case model.ConnectionTypeBambuLAN:
		client = NewBambuClient(p.ID, p.ConnectionURI, p.APIKey)
	case model.ConnectionTypeMoonraker:
		client = NewMoonrakerClient(p.ID, p.ConnectionURI)
	case model.ConnectionTypeManual:
		// No client for manual printers
		m.states[p.ID] = &model.PrinterState{
			PrinterID: p.ID,
			Status:    model.PrinterStatusOffline,
			UpdatedAt: time.Now(),
		}
		return nil
	default:
		return fmt.Errorf("unsupported connection type: %s", p.ConnectionType)
	}

	// Set up status callback
	client.SetStatusCallback(func(state *model.PrinterState) {
		m.mu.Lock()
		m.states[p.ID] = state
		m.mu.Unlock()
		slog.Info("printer status update", "printer_id", p.ID, "status", state.Status, "progress", state.Progress)

		// Broadcast state change to WebSocket clients
		m.broadcast(model.EventPrinterStateUpdated, state)
	})

	// Connect
	if err := client.Connect(); err != nil {
		slog.Error("failed to connect to printer", "printer_id", p.ID, "error", err)
		m.states[p.ID] = &model.PrinterState{
			PrinterID: p.ID,
			Status:    model.PrinterStatusOffline,
			UpdatedAt: time.Now(),
		}
		return err
	}

	m.clients[p.ID] = client

	// Get initial status
	if state, err := client.GetStatus(); err == nil {
		m.states[p.ID] = state
		// Broadcast initial state
		m.broadcast(model.EventPrinterStateUpdated, state)
	}

	slog.Info("connected to printer", "printer_id", p.ID, "type", p.ConnectionType)

	// Broadcast printer connected event
	m.broadcast(model.EventPrinterConnected, map[string]interface{}{
		"printer_id": p.ID,
	})

	return nil
}

// Disconnect closes connection to a printer.
func (m *Manager) Disconnect(id uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, ok := m.clients[id]; ok {
		client.Disconnect()
		delete(m.clients, id)
	}
	delete(m.states, id)

	// Broadcast printer disconnected event
	m.broadcast(model.EventPrinterDisconnected, map[string]interface{}{
		"printer_id": id,
	})
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

// PauseJob pauses the current print on a printer.
func (m *Manager) PauseJob(printerID uuid.UUID) error {
	m.mu.RLock()
	client, ok := m.clients[printerID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("printer not connected")
	}

	return client.PauseJob()
}

// ResumeJob resumes a paused print on a printer.
func (m *Manager) ResumeJob(printerID uuid.UUID) error {
	m.mu.RLock()
	client, ok := m.clients[printerID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("printer not connected")
	}

	return client.ResumeJob()
}

// CancelJob cancels the current print on a printer.
func (m *Manager) CancelJob(printerID uuid.UUID) error {
	m.mu.RLock()
	client, ok := m.clients[printerID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("printer not connected")
	}

	return client.CancelJob()
}

