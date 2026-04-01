package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

func TestProjectDelete_cascadesTasks(t *testing.T) {
	db := openTestDB(t)
	projectRepo := &ProjectRepository{db: db}
	taskRepo := &TaskRepository{db: db}
	ctx := context.Background()

	project := &model.Project{Name: "Cascade Test"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("Create project: %v", err)
	}

	task := &model.Task{ProjectID: project.ID, Name: "Task 1"}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	// Delete project — task should cascade
	if err := projectRepo.Delete(ctx, project.ID); err != nil {
		t.Fatalf("Delete project: %v", err)
	}

	got, err := projectRepo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetByID project: %v", err)
	}
	if got != nil {
		t.Error("project should be deleted")
	}

	gotTask, err := taskRepo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetByID task: %v", err)
	}
	if gotTask != nil {
		t.Error("task should be cascade-deleted with project")
	}
}

func TestProjectDelete_cascadesPartsAndSupplies(t *testing.T) {
	db := openTestDB(t)
	projectRepo := &ProjectRepository{db: db}
	partRepo := &PartRepository{db: db}
	supplyRepo := &ProjectSupplyRepository{db: db}
	ctx := context.Background()

	project := &model.Project{Name: "Cascade Parts"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("Create project: %v", err)
	}

	part := &model.Part{ProjectID: project.ID, Name: "Part A"}
	if err := partRepo.Create(ctx, part); err != nil {
		t.Fatalf("Create part: %v", err)
	}

	supply := &model.ProjectSupply{
		ProjectID:     project.ID,
		Name:          "Screws",
		UnitCostCents: 50,
		Quantity:      10,
	}
	if err := supplyRepo.Create(ctx, supply); err != nil {
		t.Fatalf("Create supply: %v", err)
	}

	if err := projectRepo.Delete(ctx, project.ID); err != nil {
		t.Fatalf("Delete project: %v", err)
	}

	parts, err := partRepo.ListByProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListByProject parts: %v", err)
	}
	if len(parts) != 0 {
		t.Errorf("expected 0 parts after cascade delete, got %d", len(parts))
	}

	supplies, err := supplyRepo.ListByProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListByProject supplies: %v", err)
	}
	if len(supplies) != 0 {
		t.Errorf("expected 0 supplies after cascade delete, got %d", len(supplies))
	}
}

func TestProjectDelete_blockedByPrintJobFK(t *testing.T) {
	db := openTestDB(t)
	projectRepo := &ProjectRepository{db: db}
	printJobRepo := &PrintJobRepository{db: db}
	ctx := context.Background()

	project, design := setupPrintJobTestData(t, db)

	job := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    model.PrintJobStatusQueued,
	}
	if err := printJobRepo.Create(ctx, job); err != nil {
		t.Fatalf("Create print job: %v", err)
	}

	// Direct delete should fail — print_jobs.project_id defaults to RESTRICT
	err := projectRepo.Delete(ctx, project.ID)
	if err == nil {
		t.Fatal("expected FK constraint error when deleting project with print_jobs reference, got nil")
	}
}

func TestProjectDelete_transactionalWithNullableRefs(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	projectRepo := &ProjectRepository{db: db}
	taskRepo := &TaskRepository{db: db}
	printJobRepo := &PrintJobRepository{db: db}
	saleRepo := &SaleRepository{db: db}

	// Create project
	project := &model.Project{Name: "Txn Delete Test"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("Create project: %v", err)
	}

	// Create a second project to hold print job test data
	_, design := setupPrintJobTestData(t, db)

	// Create task (should cascade)
	task := &model.Task{ProjectID: project.ID, Name: "Task 1"}
	if err := taskRepo.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	// Create print job referencing the project (nullable FK)
	job := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    model.PrintJobStatusQueued,
	}
	if err := printJobRepo.Create(ctx, job); err != nil {
		t.Fatalf("Create print job: %v", err)
	}

	// Create sale referencing the project (nullable FK)
	sale := &model.Sale{
		Channel:   "manual",
		ProjectID: &project.ID,
		Quantity:  1,
	}
	if err := saleRepo.Create(ctx, sale); err != nil {
		t.Fatalf("Create sale: %v", err)
	}

	// Perform transactional delete (same logic as ProjectService.Delete)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	for _, stmt := range []string{
		`UPDATE print_jobs SET project_id = NULL WHERE project_id = ?`,
		`UPDATE sales SET project_id = NULL WHERE project_id = ?`,
		`UPDATE etsy_receipts SET project_id = NULL WHERE project_id = ?`,
		`UPDATE order_items SET project_id = NULL WHERE project_id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, stmt, project.ID); err != nil {
			tx.Rollback()
			t.Fatalf("cleanup stmt %q: %v", stmt, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, project.ID); err != nil {
		tx.Rollback()
		t.Fatalf("delete project: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Project should be gone
	got, _ := projectRepo.GetByID(ctx, project.ID)
	if got != nil {
		t.Error("project should be deleted")
	}

	// Task should be cascade-deleted
	gotTask, _ := taskRepo.GetByID(ctx, task.ID)
	if gotTask != nil {
		t.Error("task should be cascade-deleted")
	}

	// Print job should still exist with NULL project_id
	gotJob, err := printJobRepo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID print job: %v", err)
	}
	if gotJob == nil {
		t.Fatal("print job should still exist after project delete")
	}
	if gotJob.ProjectID != nil {
		t.Errorf("print job project_id should be nil, got %v", gotJob.ProjectID)
	}

	// Sale should still exist with NULL project_id
	var saleProjectID *string
	err = db.QueryRowContext(ctx, `SELECT project_id FROM sales WHERE id = ?`, sale.ID).Scan(&saleProjectID)
	if err == sql.ErrNoRows {
		t.Fatal("sale should still exist after project delete")
	}
	if err != nil {
		t.Fatalf("query sale: %v", err)
	}
	if saleProjectID != nil {
		t.Errorf("sale project_id should be nil, got %v", *saleProjectID)
	}
}

// Verify that the NewRepositories.WithTransaction approach works for delete.
func TestProjectDelete_viaWithTransaction(t *testing.T) {
	db := openTestDB(t)
	repos := NewRepositories(db)
	ctx := context.Background()

	project := &model.Project{Name: "WithTransaction Delete"}
	if err := repos.Projects.Create(ctx, project); err != nil {
		t.Fatalf("Create project: %v", err)
	}

	task := &model.Task{ProjectID: project.ID, Name: "Task"}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	err := repos.WithTransaction(ctx, func(tx *sql.Tx) error {
		for _, stmt := range []string{
			`UPDATE print_jobs SET project_id = NULL WHERE project_id = ?`,
			`UPDATE sales SET project_id = NULL WHERE project_id = ?`,
			`UPDATE etsy_receipts SET project_id = NULL WHERE project_id = ?`,
			`UPDATE order_items SET project_id = NULL WHERE project_id = ?`,
		} {
			if _, err := tx.ExecContext(ctx, stmt, project.ID); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, project.ID)
		return err
	})
	if err != nil {
		t.Fatalf("WithTransaction delete: %v", err)
	}

	got, _ := repos.Projects.GetByID(ctx, project.ID)
	if got != nil {
		t.Error("project should be deleted")
	}

	gotTask, _ := repos.Tasks.GetByID(ctx, task.ID)
	if gotTask != nil {
		t.Error("task should be cascade-deleted")
	}
}

// Verify that deleting a project with no references works fine.
func TestProjectDelete_noReferences(t *testing.T) {
	db := openTestDB(t)
	repos := NewRepositories(db)
	ctx := context.Background()

	project := &model.Project{Name: "Lonely Project"}
	if err := repos.Projects.Create(ctx, project); err != nil {
		t.Fatalf("Create: %v", err)
	}

	err := repos.WithTransaction(ctx, func(tx *sql.Tx) error {
		for _, stmt := range []string{
			`UPDATE print_jobs SET project_id = NULL WHERE project_id = ?`,
			`UPDATE sales SET project_id = NULL WHERE project_id = ?`,
			`UPDATE etsy_receipts SET project_id = NULL WHERE project_id = ?`,
			`UPDATE order_items SET project_id = NULL WHERE project_id = ?`,
		} {
			if _, err := tx.ExecContext(ctx, stmt, project.ID); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, project.ID)
		return err
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	got, _ := repos.Projects.GetByID(ctx, project.ID)
	if got != nil {
		t.Error("project should be deleted")
	}
}

// Verify deleting a non-existent project doesn't error.
func TestProjectDelete_nonExistent(t *testing.T) {
	db := openTestDB(t)
	repos := NewRepositories(db)
	ctx := context.Background()

	err := repos.WithTransaction(ctx, func(tx *sql.Tx) error {
		for _, stmt := range []string{
			`UPDATE print_jobs SET project_id = NULL WHERE project_id = ?`,
			`UPDATE sales SET project_id = NULL WHERE project_id = ?`,
			`UPDATE etsy_receipts SET project_id = NULL WHERE project_id = ?`,
			`UPDATE order_items SET project_id = NULL WHERE project_id = ?`,
		} {
			if _, err := tx.ExecContext(ctx, stmt, uuid.New()); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, uuid.New())
		return err
	})
	if err != nil {
		t.Fatalf("delete non-existent: %v", err)
	}
}
