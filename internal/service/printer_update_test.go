package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cadimartech/rankos/internal/database"
	"github.com/cadimartech/rankos/internal/model"
	"github.com/cadimartech/rankos/internal/repository"
	"github.com/google/uuid"
)

// fakePrinterCtl records disconnect and connect calls without dialing a printer.
type fakePrinterCtl struct {
	mu           sync.Mutex
	disconnected []uuid.UUID
	connected    []*model.Printer
	connectCh    chan struct{}
}

func newFakePrinterCtl() *fakePrinterCtl {
	return &fakePrinterCtl{connectCh: make(chan struct{}, 4)}
}

func (f *fakePrinterCtl) Connect(p *model.Printer) error {
	f.mu.Lock()
	cp := *p
	f.connected = append(f.connected, &cp)
	f.mu.Unlock()
	select {
	case f.connectCh <- struct{}{}:
	default:
	}
	return nil
}

func (f *fakePrinterCtl) Disconnect(id uuid.UUID) {
	f.mu.Lock()
	f.disconnected = append(f.disconnected, id)
	f.mu.Unlock()
}

func (f *fakePrinterCtl) GetState(uuid.UUID) (*model.PrinterState, error) {
	return nil, nil
}

func (f *fakePrinterCtl) GetAllStates() map[uuid.UUID]*model.PrinterState {
	return map[uuid.UUID]*model.PrinterState{}
}

func (f *fakePrinterCtl) Send(uuid.UUID, model.Command) error { return nil }

func (f *fakePrinterCtl) counts() (disconnects int, connects int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.disconnected), len(f.connected)
}

func (f *fakePrinterCtl) lastConnected() *model.Printer {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.connected) == 0 {
		return nil
	}
	return f.connected[len(f.connected)-1]
}

func (f *fakePrinterCtl) waitConnect(t *testing.T) *model.Printer {
	t.Helper()
	select {
	case <-f.connectCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for printer reconnect")
	}
	p := f.lastConnected()
	if p == nil {
		t.Fatal("connect was signaled but no printer was recorded")
	}
	return p
}

func newPrinterServiceForTest(t *testing.T, ctl printerController) (*PrinterService, *repository.PrinterRepository) {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	repo := repository.NewRepositories(db).Printers
	return &PrinterService{repo: repo, manager: ctl}, repo
}

func TestPrinterUpdate_NameChangeDoesNotReconnect(t *testing.T) {
	ctl := newFakePrinterCtl()
	svc, repo := newPrinterServiceForTest(t, ctl)
	ctx := context.Background()

	p := &model.Printer{
		Name:           "P1S",
		ConnectionType: model.ConnectionTypeBambuLAN,
		ConnectionURI:  "192.168.1.20",
		APIKey:         "12345678",
		SerialNumber:   "22E8AA000000001",
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatal(err)
	}

	p.Name = "P1S Rack"
	if err := svc.Update(ctx, p); err != nil {
		t.Fatal(err)
	}

	time.Sleep(30 * time.Millisecond)
	disconnects, connects := ctl.counts()
	if disconnects != 0 || connects != 0 {
		t.Fatalf("name change reconnected: disconnects=%d connects=%d", disconnects, connects)
	}
	stored, err := repo.GetByID(ctx, p.ID)
	if err != nil || stored == nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Name != "P1S Rack" {
		t.Fatalf("name = %q", stored.Name)
	}
	if stored.APIKey != "12345678" {
		t.Fatalf("api key changed: %q", stored.APIKey)
	}
}

func TestPrinterUpdate_ConnectionChangeReconnectsAndKeepsBlankAccessCode(t *testing.T) {
	ctl := newFakePrinterCtl()
	svc, repo := newPrinterServiceForTest(t, ctl)
	ctx := context.Background()

	p := &model.Printer{
		Name:           "P1S",
		ConnectionType: model.ConnectionTypeBambuLAN,
		ConnectionURI:  "192.168.1.20",
		APIKey:         "12345678",
		SerialNumber:   "WRONGSERIAL",
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatal(err)
	}

	// Blank access code means "keep the stored one".
	p.APIKey = ""
	p.ConnectionURI = "192.168.1.55"
	p.SerialNumber = "22E8AA000000001"
	if err := svc.Update(ctx, p); err != nil {
		t.Fatal(err)
	}

	dialed := ctl.waitConnect(t)
	if dialed.ConnectionURI != "192.168.1.55" {
		t.Fatalf("dialed host = %q", dialed.ConnectionURI)
	}
	if dialed.SerialNumber != "22E8AA000000001" {
		t.Fatalf("dialed serial = %q", dialed.SerialNumber)
	}
	if dialed.APIKey != "12345678" {
		t.Fatalf("dialed access code = %q, want stored code", dialed.APIKey)
	}
	disconnects, connects := ctl.counts()
	if disconnects != 1 || connects != 1 {
		t.Fatalf("disconnects=%d connects=%d", disconnects, connects)
	}

	stored, err := repo.GetByID(ctx, p.ID)
	if err != nil || stored == nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.APIKey != "12345678" {
		t.Fatalf("stored access code = %q", stored.APIKey)
	}
	if stored.SerialNumber != "22E8AA000000001" || stored.ConnectionURI != "192.168.1.55" {
		t.Fatalf("stored connection = %s %s", stored.ConnectionURI, stored.SerialNumber)
	}
}

func TestPrinterUpdate_RedactedAccessCodeIsNotSaved(t *testing.T) {
	ctl := newFakePrinterCtl()
	svc, repo := newPrinterServiceForTest(t, ctl)
	ctx := context.Background()

	p := &model.Printer{
		Name:           "P1S",
		ConnectionType: model.ConnectionTypeBambuLAN,
		ConnectionURI:  "192.168.1.20",
		APIKey:         "12345678",
		SerialNumber:   "22E8AA000000001",
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatal(err)
	}

	p.APIKey = model.RedactedSecret
	if err := svc.Update(ctx, p); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	disconnects, connects := ctl.counts()
	if disconnects != 0 || connects != 0 {
		t.Fatalf("redacted key triggered reconnect: disconnects=%d connects=%d", disconnects, connects)
	}
	stored, _ := repo.GetByID(ctx, p.ID)
	if stored.APIKey != "12345678" {
		t.Fatalf("stored access code = %q", stored.APIKey)
	}
}

func TestPrinterUpdate_NewAccessCodeReconnects(t *testing.T) {
	ctl := newFakePrinterCtl()
	svc, repo := newPrinterServiceForTest(t, ctl)
	ctx := context.Background()

	p := &model.Printer{
		Name:           "P1S",
		ConnectionType: model.ConnectionTypeBambuLAN,
		ConnectionURI:  "192.168.1.20",
		APIKey:         "12345678",
		SerialNumber:   "22E8AA000000001",
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatal(err)
	}

	p.APIKey = "87654321"
	if err := svc.Update(ctx, p); err != nil {
		t.Fatal(err)
	}
	dialed := ctl.waitConnect(t)
	if dialed.APIKey != "87654321" {
		t.Fatalf("dialed access code = %q", dialed.APIKey)
	}
	stored, _ := repo.GetByID(ctx, p.ID)
	if stored.APIKey != "87654321" {
		t.Fatalf("stored access code = %q", stored.APIKey)
	}
}

func TestPrinterUpdate_ManualDoesNotConnect(t *testing.T) {
	ctl := newFakePrinterCtl()
	svc, repo := newPrinterServiceForTest(t, ctl)
	ctx := context.Background()

	p := &model.Printer{
		Name:           "P1S",
		ConnectionType: model.ConnectionTypeBambuLAN,
		ConnectionURI:  "192.168.1.20",
		APIKey:         "12345678",
		SerialNumber:   "22E8AA000000001",
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatal(err)
	}

	p.ConnectionType = model.ConnectionTypeManual
	if err := svc.Update(ctx, p); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	disconnects, connects := ctl.counts()
	if disconnects != 1 {
		t.Fatalf("disconnects=%d, want the old session dropped", disconnects)
	}
	if connects != 0 {
		t.Fatalf("manual printer was dialed")
	}
}
