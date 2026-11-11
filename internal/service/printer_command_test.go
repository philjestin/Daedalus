package service

import (
	"context"
	"testing"

	"github.com/cadimartech/rankos/internal/database"
	"github.com/cadimartech/rankos/internal/model"
	"github.com/cadimartech/rankos/internal/printer"
	"github.com/cadimartech/rankos/internal/realtime"
	"github.com/cadimartech/rankos/internal/repository"
	"github.com/cadimartech/rankos/internal/storage"
)

func TestExternalPauseDoesNotFailJob(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	repos := repository.NewRepositories(db)
	hub := realtime.NewHub()
	mgr := printer.NewManager()
	store := storage.NewLocalStorage(t.TempDir())
	services := NewServices(repos, store, mgr, hub)
	ctx := context.Background()

	job := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 0, true)
	machine := createTestPrinter(t, repos, "P2S-01")
	job.PrinterID = &machine.ID
	if err := repos.PrintJobs.Update(ctx, job); err != nil {
		t.Fatal(err)
	}
	printing := model.PrintJobStatusPrinting
	if err := repos.PrintJobs.AppendEvent(ctx, model.NewJobEvent(job.ID, model.JobEventStarted, &printing).WithPrinter(machine.ID)); err != nil {
		t.Fatal(err)
	}

	services.PrintJobs.handlePrinterStatusChange(
		&model.PrinterState{PrinterID: machine.ID, Status: model.PrinterStatusPaused, GcodeState: "PAUSE"},
		&model.PrinterState{PrinterID: machine.ID, Status: model.PrinterStatusPrinting},
	)
	got, err := repos.PrintJobs.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.PrintJobStatusPaused {
		t.Fatalf("after external pause status = %s, want paused", got.Status)
	}

	services.PrintJobs.handlePrinterStatusChange(
		&model.PrinterState{PrinterID: machine.ID, Status: model.PrinterStatusError, GcodeState: "PAUSE", PrintError: 1},
		&model.PrinterState{PrinterID: machine.ID, Status: model.PrinterStatusPaused},
	)
	got, err = repos.PrintJobs.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.PrintJobStatusPaused {
		t.Fatalf("pause then error status = %s, want paused", got.Status)
	}

	services.PrintJobs.handlePrinterStatusChange(
		&model.PrinterState{PrinterID: machine.ID, Status: model.PrinterStatusPrinting, GcodeState: "RUNNING"},
		&model.PrinterState{PrinterID: machine.ID, Status: model.PrinterStatusPaused},
	)
	got, err = repos.PrintJobs.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.PrintJobStatusPrinting {
		t.Fatalf("external resume status = %s, want printing", got.Status)
	}

	if err := services.PrintJobs.NotePrinterCommand(ctx, machine.ID, model.CmdStop, model.ActorUser); err != nil {
		t.Fatal(err)
	}
	got, err = repos.PrintJobs.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.PrintJobStatusCancelled {
		t.Fatalf("stop status = %s, want cancelled", got.Status)
	}
}

func TestPrintingToErrorStillFailsJob(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	repos := repository.NewRepositories(db)
	hub := realtime.NewHub()
	mgr := printer.NewManager()
	store := storage.NewLocalStorage(t.TempDir())
	services := NewServices(repos, store, mgr, hub)
	ctx := context.Background()

	job := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 0, true)
	machine := createTestPrinter(t, repos, "H2S-01")
	job.PrinterID = &machine.ID
	if err := repos.PrintJobs.Update(ctx, job); err != nil {
		t.Fatal(err)
	}
	printing := model.PrintJobStatusPrinting
	if err := repos.PrintJobs.AppendEvent(ctx, model.NewJobEvent(job.ID, model.JobEventStarted, &printing).WithPrinter(machine.ID)); err != nil {
		t.Fatal(err)
	}

	services.PrintJobs.handlePrinterStatusChange(
		&model.PrinterState{PrinterID: machine.ID, Status: model.PrinterStatusError},
		&model.PrinterState{PrinterID: machine.ID, Status: model.PrinterStatusPrinting},
	)
	got, err := repos.PrintJobs.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.PrintJobStatusFailed {
		t.Fatalf("status = %s, want failed", got.Status)
	}
}
