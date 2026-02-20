package price

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	domain "github.com/ahmethakanbesel/finance-api/internal/price"
)

const dateFormat = "2006-01-02"

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SavePrices(ctx context.Context, prices []domain.Price) (int64, error) {
	if len(prices) == 0 {
		return 0, nil
	}

	const batchSize = 500
	var total int64

	for i := 0; i < len(prices); i += batchSize {
		end := i + batchSize
		if end > len(prices) {
			end = len(prices)
		}
		batch := prices[i:end]

		placeholders := make([]string, len(batch))
		args := make([]any, 0, len(batch)*5)
		for j, p := range batch {
			placeholders[j] = "(?, ?, ?, ?, ?)"
			args = append(args, string(p.Source), p.Symbol, p.Date.Format(dateFormat), p.ClosePrice, string(p.Currency))
		}

		query := fmt.Sprintf( //nolint:gosec // placeholders are not user input
			"INSERT OR IGNORE INTO prices (source, symbol, date, close_price, currency) VALUES %s",
			strings.Join(placeholders, ", "),
		)

		res, err := r.db.ExecContext(ctx, query, args...)
		if err != nil {
			return total, fmt.Errorf("save prices: %w", err)
		}

		n, _ := res.RowsAffected()
		total += n
	}

	return total, nil
}

func (r *Repository) ListPrices(ctx context.Context, source domain.Source, symbol string, from, to time.Time) ([]domain.Price, error) {
	const query = `SELECT id, source, symbol, date, close_price, currency, created_at
		FROM prices
		WHERE source = ? AND symbol = ? AND date >= ? AND date <= ?
		ORDER BY date ASC`

	rows, err := r.db.QueryContext(ctx, query,
		string(source), symbol,
		from.Format(dateFormat), to.Format(dateFormat),
	)
	if err != nil {
		return nil, fmt.Errorf("list prices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var prices []domain.Price
	for rows.Next() {
		var p domain.Price
		var src, cur, dateStr, createdStr string
		if err := rows.Scan(&p.ID, &src, &p.Symbol, &dateStr, &p.ClosePrice, &cur, &createdStr); err != nil {
			return nil, fmt.Errorf("scan price: %w", err)
		}
		p.Source = domain.Source(src)
		p.Currency = domain.Currency(cur)
		p.Date, _ = time.Parse(dateFormat, dateStr)
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		prices = append(prices, p)
	}

	return prices, rows.Err()
}

func (r *Repository) ExistingDates(ctx context.Context, source domain.Source, symbol string, from, to time.Time) (map[time.Time]bool, error) {
	const query = `SELECT date FROM prices
		WHERE source = ? AND symbol = ? AND date >= ? AND date <= ?`

	rows, err := r.db.QueryContext(ctx, query,
		string(source), symbol,
		from.Format(dateFormat), to.Format(dateFormat),
	)
	if err != nil {
		return nil, fmt.Errorf("existing dates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	dates := make(map[time.Time]bool)
	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err != nil {
			return nil, fmt.Errorf("scan date: %w", err)
		}
		t, _ := time.Parse(dateFormat, dateStr)
		dates[t] = true
	}

	return dates, rows.Err()
}
