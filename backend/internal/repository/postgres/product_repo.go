package postgres

import (
	"context"
	"errors"
	"fmt"

	"ecommerce-backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

type ProductRepo struct {
	db *DB
}

func NewProductRepo(db *DB) *ProductRepo {
	return &ProductRepo{db: db}
}

func (r *ProductRepo) GetByID(ctx context.Context, id int64) (*domain.Product, error) {
	query := `
		SELECT id, sku, name, description, price, category, stock_quantity, created_at, updated_at
		FROM products
		WHERE id = $1
	`
	var p domain.Product
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.SKU,
		&p.Name,
		&p.Description,
		&p.Price,
		&p.Category,
		&p.StockQuantity,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query product by id: %w", err)
	}
	return &p, nil
}

func (r *ProductRepo) List(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var rows pgx.Rows
	var err error

	if filter.Category != "" {
		query := `
			SELECT id, sku, name, description, price, category, stock_quantity, created_at, updated_at
			FROM products
			WHERE category = $1
			ORDER BY id ASC
			LIMIT $2 OFFSET $3
		`
		rows, err = r.db.Pool.Query(ctx, query, filter.Category, limit, filter.Offset)
	} else {
		query := `
			SELECT id, sku, name, description, price, category, stock_quantity, created_at, updated_at
			FROM products
			ORDER BY id ASC
			LIMIT $1 OFFSET $2
		`
		rows, err = r.db.Pool.Query(ctx, query, limit, filter.Offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	products := make([]domain.Product, 0, limit)
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(
			&p.ID,
			&p.SKU,
			&p.Name,
			&p.Description,
			&p.Price,
			&p.Category,
			&p.StockQuantity,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}

	return products, nil
}

func (r *ProductRepo) GetCategories(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT category FROM products ORDER BY category ASC`
	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}
	return categories, nil
}
