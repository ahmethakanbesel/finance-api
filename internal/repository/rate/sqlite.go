package rate

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	domain "github.com/ahmethakanbesel/finance-api/internal/rate"
)

const dateFormat = "2006-01-02"

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SaveRates(ctx context.Context, rates []domain.Rate) (int64, error) {
	if len(rates) == 0 {
		return 0, nil
	}

	const batchSize = 500
	var total int64

	for i := 0; i < len(rates); i += batchSize {
		end := i + batchSize
		if end > len(rates) {
			end = len(rates)
		}
		batch := rates[i:end]

		placeholders := make([]string, len(batch))
		args := make([]any, 0, len(batch)*3)
		for j, rate := range batch {
			placeholders[j] = "(?, ?, ?)"
			args = append(args, rate.Pair, rate.Date.Format(dateFormat), rate.Rate)
		}

		query := fmt.Sprintf( //nolint:gosec // placeholders are not user input
			"INSERT OR IGNORE INTO exchange_rates (pair, date, rate) VALUES %s",
			strings.Join(placeholders, ", "),
		)

		res, err := r.db.ExecContext(ctx, query, args...)
		if err != nil {
			return total, fmt.Errorf("save rates: %w", err)
		}

		n, _ := res.RowsAffected()
		total += n
	}

	return total, nil
}

func (r *Repository) ListRates(ctx context.Context, pair string, from, to time.Time) ([]domain.Rate, error) {
	const query = `SELECT id, pair, date, rate, created_at
		FROM exchange_rates
		WHERE pair = ? AND date >= ? AND date <= ?
		ORDER BY date ASC`

	rows, err := r.db.QueryContext(ctx, query, pair, from.Format(dateFormat), to.Format(dateFormat))
	if err != nil {
		return nil, fmt.Errorf("list rates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var rates []domain.Rate
	for rows.Next() {
		var rate domain.Rate
		var dateStr, createdStr string
		if err := rows.Scan(&rate.ID, &rate.Pair, &dateStr, &rate.Rate, &createdStr); err != nil {
			return nil, fmt.Errorf("scan rate: %w", err)
		}
		rate.Date, _ = time.Parse(dateFormat, dateStr)
		rate.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		rates = append(rates, rate)
	}

	return rates, rows.Err()
}

func (r *Repository) ExistingDates(ctx context.Context, pair string, from, to time.Time) (map[time.Time]bool, error) {
	const query = `SELECT date FROM exchange_rates
		WHERE pair = ? AND date >= ? AND date <= ?`

	rows, err := r.db.QueryContext(ctx, query, pair, from.Format(dateFormat), to.Format(dateFormat))
	if err != nil {
		return nil, fmt.Errorf("existing rate dates: %w", err)
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
