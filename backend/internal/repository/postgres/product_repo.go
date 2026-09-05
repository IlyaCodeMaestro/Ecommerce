package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// Search performs high-speed Full-Text Search with GIN index and multi-faceted filtering
func (r *ProductRepo) Search(ctx context.Context, filter domain.ProductFilter) (*domain.ProductSearchResult, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var whereClauses []string
	var args []interface{}
	argIdx := 1
	var queryArgIdx int

	// 1. Full-Text Search via GIN Inverted Index
	if strings.TrimSpace(filter.Query) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(
			"to_tsvector('english', name || ' ' || coalesce(description, '')) @@ plainto_tsquery('english', $%d)",
			argIdx,
		))
		args = append(args, strings.TrimSpace(filter.Query))
		queryArgIdx = argIdx
		argIdx++
	}

	// 2. Category Filter
	if strings.TrimSpace(filter.Category) != "" && filter.Category != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.Category))
		argIdx++
	}

	// 3. Price Bounds
	if filter.MinPrice != nil && *filter.MinPrice >= 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("price >= $%d", argIdx))
		args = append(args, *filter.MinPrice)
		argIdx++
	}
	if filter.MaxPrice != nil && *filter.MaxPrice > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("price <= $%d", argIdx))
		args = append(args, *filter.MaxPrice)
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 4. Count total matching rows
	countSQL := fmt.Sprintf("SELECT count(*) FROM products %s", whereSQL)
	var totalCount int64
	if err := r.db.Pool.QueryRow(ctx, countSQL, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	// 5. Determine Ordering
	var orderBy string
	if queryArgIdx > 0 && (filter.SortBy == "relevance" || filter.SortBy == "") {
		orderBy = fmt.Sprintf(
			"ORDER BY ts_rank(to_tsvector('english', name || ' ' || coalesce(description, '')), plainto_tsquery('english', $%d)) DESC, id ASC",
			queryArgIdx,
		)
	} else {
		switch filter.SortBy {
		case "price_asc":
			orderBy = "ORDER BY price ASC, id ASC"
		case "price_desc":
			orderBy = "ORDER BY price DESC, id ASC"
		case "newest":
			orderBy = "ORDER BY created_at DESC, id ASC"
		default:
			orderBy = "ORDER BY id ASC"
		}
	}

	// 6. Query matching products with pagination
	dataSQL := fmt.Sprintf(`
		SELECT id, sku, name, description, price, category, stock_quantity, created_at, updated_at
		FROM products
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereSQL, orderBy, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, dataSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query search products: %w", err)
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
			return nil, fmt.Errorf("failed to scan search product: %w", err)
		}
		products = append(products, p)
	}

	return &domain.ProductSearchResult{
		Total:    totalCount,
		Products: products,
		Limit:    limit,
		Offset:   offset,
	}, nil
}

func (r *ProductRepo) List(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, error) {
	res, err := r.Search(ctx, filter)
	if err != nil {
		return nil, err
	}
	return res.Products, nil
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
