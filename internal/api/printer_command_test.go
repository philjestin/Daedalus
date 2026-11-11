package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cadimartech/rankos/internal/model"
	"github.com/google/uuid"
)

func TestPrinterCommand_badBodyAndUnknownPrinter(t *testing.T) {
	env := newTestEnv(t)
	id := uuid.New()

	tests := []struct {
		name string
		body string
		code int
	}{
		{name: "gcode field", body: `{"kind":"pause","gcode":"G28 X0"}`, code: http.StatusBadRequest},
		{name: "gcode kind", body: `{"kind":"gcode_line"}`, code: http.StatusBadRequest},
		{name: "speed out of range", body: `{"kind":"set_speed","level":9}`, code: http.StatusBadRequest},
		{name: "unknown printer", body: `{"kind":"pause"}`, code: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/printers/"+id.String()+"/commands", bytes.NewBufferString(tt.body))
			rr := httptest.NewRecorder()
			env.handler.ServeHTTP(rr, req)
			if rr.Code != tt.code {
				t.Fatalf("status = %d, want %d, body %s", rr.Code, tt.code, rr.Body.String())
			}
		})
	}
}

func TestPrinterCommand_offlinePrinter(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	p := &model.Printer{
		Name:           "P2S-01",
		Model:          "P2S",
		ConnectionType: model.ConnectionTypeManual,
	}
	if err := env.services.Printers.Create(ctx, p); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/printers/"+p.ID.String()+"/commands", bytes.NewBufferString(`{"kind":"stop"}`))
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, body %s", rr.Code, rr.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] == "" {
		t.Fatalf("body %s", rr.Body.String())
	}
}
