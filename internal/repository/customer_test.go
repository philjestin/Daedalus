package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

func TestCustomerRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDB(t)
	repo := &CustomerRepository{db: db}
	ctx := context.Background()

	customer := &model.Customer{
		Name:    "Alice Smith",
		Email:   "alice@example.com",
		Company: "Acme Corp",
		Phone:   "555-1234",
		Notes:   "VIP customer",
	}

	if err := repo.Create(ctx, customer); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if customer.ID == uuid.Nil {
		t.Error("Create should set customer ID")
	}
	if customer.CreatedAt.IsZero() {
		t.Error("Create should set CreatedAt")
	}
	if customer.UpdatedAt.IsZero() {
		t.Error("Create should set UpdatedAt")
	}

	got, err := repo.GetByID(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil")
	}

	if got.Name != "Alice Smith" {
		t.Errorf("Name = %q, want %q", got.Name, "Alice Smith")
	}
	if got.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "alice@example.com")
	}
	if got.Company != "Acme Corp" {
		t.Errorf("Company = %q, want %q", got.Company, "Acme Corp")
	}
	if got.Phone != "555-1234" {
		t.Errorf("Phone = %q, want %q", got.Phone, "555-1234")
	}
	if got.Notes != "VIP customer" {
		t.Errorf("Notes = %q, want %q", got.Notes, "VIP customer")
	}
}

func TestCustomerRepository_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := &CustomerRepository{db: db}
	ctx := context.Background()

	got, err := repo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("GetByID should return nil for non-existent customer")
	}
}

func TestCustomerRepository_GetByEmail(t *testing.T) {
	db := openTestDB(t)
	repo := &CustomerRepository{db: db}
	ctx := context.Background()

	customer := &model.Customer{
		Name:  "Bob Johnson",
		Email: "bob@example.com",
	}
	if err := repo.Create(ctx, customer); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByEmail(ctx, "bob@example.com")
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetByEmail returned nil for existing email")
	}
	if got.Name != "Bob Johnson" {
		t.Errorf("Name = %q, want %q", got.Name, "Bob Johnson")
	}
	if got.ID != customer.ID {
		t.Errorf("ID = %v, want %v", got.ID, customer.ID)
	}

	// Non-existent email should return nil
	missing, err := repo.GetByEmail(ctx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("GetByEmail for missing email failed: %v", err)
	}
	if missing != nil {
		t.Error("GetByEmail should return nil for non-existent email")
	}
}

func TestCustomerRepository_List(t *testing.T) {
	db := openTestDB(t)
	repo := &CustomerRepository{db: db}
	ctx := context.Background()

	names := []string{"Alice", "Bob", "Charlie"}
	for _, name := range names {
		c := &model.Customer{Name: name}
		if err := repo.Create(ctx, c); err != nil {
			t.Fatalf("Create %q failed: %v", name, err)
		}
	}

	customers, err := repo.List(ctx, model.CustomerFilters{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(customers) != 3 {
		t.Errorf("len(customers) = %d, want 3", len(customers))
	}

	// Should be ordered by name ASC
	if customers[0].Name != "Alice" {
		t.Errorf("First customer = %q, want %q", customers[0].Name, "Alice")
	}
	if customers[1].Name != "Bob" {
		t.Errorf("Second customer = %q, want %q", customers[1].Name, "Bob")
	}
	if customers[2].Name != "Charlie" {
		t.Errorf("Third customer = %q, want %q", customers[2].Name, "Charlie")
	}
}

func TestCustomerRepository_List_Search(t *testing.T) {
	db := openTestDB(t)
	repo := &CustomerRepository{db: db}
	ctx := context.Background()

	customers := []*model.Customer{
		{Name: "Alice Smith", Email: "alice@example.com", Company: "Acme Corp"},
		{Name: "Bob Johnson", Email: "bob@widgets.com", Company: "Widget Inc"},
		{Name: "Charlie Brown", Email: "charlie@acme.com", Company: "Acme Corp"},
	}
	for _, c := range customers {
		if err := repo.Create(ctx, c); err != nil {
			t.Fatalf("Create %q failed: %v", c.Name, err)
		}
	}

	// Search by name
	results, err := repo.List(ctx, model.CustomerFilters{Search: "Alice"})
	if err != nil {
		t.Fatalf("List with name search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Search 'Alice': len = %d, want 1", len(results))
	}

	// Search by email
	results, err = repo.List(ctx, model.CustomerFilters{Search: "widgets"})
	if err != nil {
		t.Fatalf("List with email search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Search 'widgets': len = %d, want 1", len(results))
	}

	// Search by company (should match both Acme customers)
	results, err = repo.List(ctx, model.CustomerFilters{Search: "Acme"})
	if err != nil {
		t.Fatalf("List with company search failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Search 'Acme': len = %d, want 2", len(results))
	}

	// Search with no matches
	results, err = repo.List(ctx, model.CustomerFilters{Search: "Zzzzz"})
	if err != nil {
		t.Fatalf("List with no-match search failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search 'Zzzzz': len = %d, want 0", len(results))
	}
}

func TestCustomerRepository_Update(t *testing.T) {
	db := openTestDB(t)
	repo := &CustomerRepository{db: db}
	ctx := context.Background()

	customer := &model.Customer{
		Name:  "Original Name",
		Email: "original@example.com",
	}
	if err := repo.Create(ctx, customer); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	customer.Name = "Updated Name"
	customer.Email = "updated@example.com"
	customer.Company = "New Company"
	customer.Phone = "555-9999"
	customer.Notes = "Updated notes"

	if err := repo.Update(ctx, customer); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := repo.GetByID(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}

	if got.Name != "Updated Name" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated Name")
	}
	if got.Email != "updated@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "updated@example.com")
	}
	if got.Company != "New Company" {
		t.Errorf("Company = %q, want %q", got.Company, "New Company")
	}
	if got.Phone != "555-9999" {
		t.Errorf("Phone = %q, want %q", got.Phone, "555-9999")
	}
	if got.Notes != "Updated notes" {
		t.Errorf("Notes = %q, want %q", got.Notes, "Updated notes")
	}
}

func TestCustomerRepository_Delete(t *testing.T) {
	db := openTestDB(t)
	repo := &CustomerRepository{db: db}
	ctx := context.Background()

	customer := &model.Customer{Name: "To Delete"}
	if err := repo.Create(ctx, customer); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := repo.Delete(ctx, customer.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, err := repo.GetByID(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetByID after delete failed: %v", err)
	}
	if got != nil {
		t.Error("Customer should be nil after delete")
	}
}
