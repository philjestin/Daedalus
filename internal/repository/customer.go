package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

// CustomerRepository handles customer database operations.
type CustomerRepository struct {
	db *sql.DB
}

// Create inserts a new customer.
func (r *CustomerRepository) Create(ctx context.Context, customer *model.Customer) error {
	customer.ID = uuid.New()
	customer.CreatedAt = time.Now()
	customer.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO customers (id, name, email, company, phone, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, customer.ID, customer.Name, customer.Email, customer.Company, customer.Phone, customer.Notes, customer.CreatedAt, customer.UpdatedAt)
	return err
}

// GetByID retrieves a customer by ID.
func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Customer, error) {
	var c model.Customer
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, email, company, phone, notes, created_at, updated_at
		FROM customers WHERE id = ?
	`, id)
	err := scanRow(row, &c.ID, &c.Name, &c.Email, &c.Company, &c.Phone, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

// GetByEmail retrieves a customer by email.
func (r *CustomerRepository) GetByEmail(ctx context.Context, email string) (*model.Customer, error) {
	var c model.Customer
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, email, company, phone, notes, created_at, updated_at
		FROM customers WHERE email = ?
	`, email)
	err := scanRow(row, &c.ID, &c.Name, &c.Email, &c.Company, &c.Phone, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

// List retrieves customers with optional search filtering.
func (r *CustomerRepository) List(ctx context.Context, filters model.CustomerFilters) ([]model.Customer, error) {
	query := `
		SELECT id, name, email, company, phone, notes, created_at, updated_at
		FROM customers WHERE 1=1
	`
	args := []interface{}{}

	if filters.Search != "" {
		query += " AND (name LIKE ? OR email LIKE ? OR company LIKE ?)"
		search := "%" + filters.Search + "%"
		args = append(args, search, search, search)
	}

	query += " ORDER BY name ASC"

	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}
	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []model.Customer
	for rows.Next() {
		var c model.Customer
		if err := scanRow(rows, &c.ID, &c.Name, &c.Email, &c.Company, &c.Phone, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, rows.Err()
}

// Update updates a customer.
func (r *CustomerRepository) Update(ctx context.Context, customer *model.Customer) error {
	customer.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE customers SET name = ?, email = ?, company = ?, phone = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`, customer.Name, customer.Email, customer.Company, customer.Phone, customer.Notes, customer.UpdatedAt, customer.ID)
	return err
}

// Delete removes a customer.
func (r *CustomerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM customers WHERE id = ?`, id)
	return err
}
