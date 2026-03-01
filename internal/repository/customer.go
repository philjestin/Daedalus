package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

// CustomerRepository handles customer database operations.
type CustomerRepository struct {
	db *sql.DB
}

// marshalAddress marshals an Address to JSON string for storage.
func marshalAddress(addr *model.Address) *string {
	if addr == nil {
		return nil
	}
	data, err := json.Marshal(addr)
	if err != nil {
		return nil
	}
	s := string(data)
	return &s
}

// unmarshalAddress unmarshals a JSON string to an Address.
func unmarshalAddress(s *string) *model.Address {
	if s == nil || *s == "" {
		return nil
	}
	var addr model.Address
	if err := json.Unmarshal([]byte(*s), &addr); err != nil {
		return nil
	}
	return &addr
}

// Create inserts a new customer.
func (r *CustomerRepository) Create(ctx context.Context, customer *model.Customer) error {
	customer.ID = uuid.New()
	customer.CreatedAt = time.Now()
	customer.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO customers (id, name, email, company, phone, notes, billing_address_json, shipping_address_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, customer.ID, customer.Name, customer.Email, customer.Company, customer.Phone, customer.Notes,
		marshalAddress(customer.BillingAddress), marshalAddress(customer.ShippingAddress),
		customer.CreatedAt, customer.UpdatedAt)
	return err
}

// GetByID retrieves a customer by ID.
func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Customer, error) {
	var c model.Customer
	var billingJSON, shippingJSON *string
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, email, company, phone, notes, billing_address_json, shipping_address_json, created_at, updated_at
		FROM customers WHERE id = ?
	`, id)
	err := scanRow(row, &c.ID, &c.Name, &c.Email, &c.Company, &c.Phone, &c.Notes, &billingJSON, &shippingJSON, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.BillingAddress = unmarshalAddress(billingJSON)
	c.ShippingAddress = unmarshalAddress(shippingJSON)
	return &c, nil
}

// GetByEmail retrieves a customer by email.
func (r *CustomerRepository) GetByEmail(ctx context.Context, email string) (*model.Customer, error) {
	var c model.Customer
	var billingJSON, shippingJSON *string
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, email, company, phone, notes, billing_address_json, shipping_address_json, created_at, updated_at
		FROM customers WHERE email = ?
	`, email)
	err := scanRow(row, &c.ID, &c.Name, &c.Email, &c.Company, &c.Phone, &c.Notes, &billingJSON, &shippingJSON, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.BillingAddress = unmarshalAddress(billingJSON)
	c.ShippingAddress = unmarshalAddress(shippingJSON)
	return &c, nil
}

// List retrieves customers with optional search filtering.
func (r *CustomerRepository) List(ctx context.Context, filters model.CustomerFilters) ([]model.Customer, error) {
	query := `
		SELECT id, name, email, company, phone, notes, billing_address_json, shipping_address_json, created_at, updated_at
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
		var billingJSON, shippingJSON *string
		if err := scanRow(rows, &c.ID, &c.Name, &c.Email, &c.Company, &c.Phone, &c.Notes, &billingJSON, &shippingJSON, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.BillingAddress = unmarshalAddress(billingJSON)
		c.ShippingAddress = unmarshalAddress(shippingJSON)
		customers = append(customers, c)
	}
	return customers, rows.Err()
}

// Update updates a customer.
func (r *CustomerRepository) Update(ctx context.Context, customer *model.Customer) error {
	customer.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE customers SET name = ?, email = ?, company = ?, phone = ?, notes = ?,
			billing_address_json = ?, shipping_address_json = ?, updated_at = ?
		WHERE id = ?
	`, customer.Name, customer.Email, customer.Company, customer.Phone, customer.Notes,
		marshalAddress(customer.BillingAddress), marshalAddress(customer.ShippingAddress),
		customer.UpdatedAt, customer.ID)
	return err
}

// Delete removes a customer.
func (r *CustomerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM customers WHERE id = ?`, id)
	return err
}
