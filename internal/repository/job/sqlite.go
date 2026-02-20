package job

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ahmethakanbesel/finance-api/internal/apperror"
	domain "github.com/ahmethakanbesel/finance-api/internal/job"
)

const dateFormat = "2006-01-02"

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, j *domain.Job) error {
	const query = `INSERT INTO jobs (source, symbol, start_date, end_date, status)
		VALUES (?, ?, ?, ?, ?)`

	res, err := r.db.ExecContext(ctx, query,
		j.Source, j.Symbol,
		j.StartDate.Format(dateFormat), j.EndDate.Format(dateFormat),
		string(j.Status),
	)
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}

	j.ID, _ = res.LastInsertId()
	j.CreatedAt = time.Now().UTC()
	j.UpdatedAt = j.CreatedAt
	return nil
}

func (r *Repository) Update(ctx context.Context, j *domain.Job) error {
	const query = `UPDATE jobs SET status = ?, error = ?, records_count = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, string(j.Status), j.Error, j.RecordsCount, j.ID)
	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}
	j.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *Repository) Get(ctx context.Context, id int64) (*domain.Job, error) {
	const query = `SELECT id, source, symbol, start_date, end_date,
		status, error, records_count, created_at, updated_at
		FROM jobs WHERE id = ?`

	j := &domain.Job{}
	var startStr, endStr, status string
	var createdStr, updatedStr string
	var dbErr sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&j.ID, &j.Source, &j.Symbol,
		&startStr, &endStr, &status, &dbErr,
		&j.RecordsCount, &createdStr, &updatedStr,
	)
	if err == sql.ErrNoRows {
		return nil, apperror.New(apperror.NotFound, "job not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}

	j.Status = domain.Status(status)
	if dbErr.Valid {
		j.Error = dbErr.String
	}
	j.StartDate, _ = time.Parse(dateFormat, startStr)
	j.EndDate, _ = time.Parse(dateFormat, endStr)
	j.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	j.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return j, nil
}

func (r *Repository) List(ctx context.Context, source, symbol string) ([]domain.Job, error) {
	query := `SELECT id, source, symbol, start_date, end_date,
		status, error, records_count, created_at, updated_at
		FROM jobs WHERE 1=1`

	var args []any
	if source != "" {
		query += " AND source = ?"
		args = append(args, source)
	}
	if symbol != "" {
		query += " AND symbol = ?"
		args = append(args, symbol)
	}
	query += " ORDER BY id DESC LIMIT 100"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var jobs []domain.Job
	for rows.Next() {
		var j domain.Job
		var startStr, endStr, status, createdStr, updatedStr string
		var dbErr sql.NullString

		if err := rows.Scan(
			&j.ID, &j.Source, &j.Symbol,
			&startStr, &endStr, &status, &dbErr,
			&j.RecordsCount, &createdStr, &updatedStr,
		); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}

		j.Status = domain.Status(status)
		if dbErr.Valid {
			j.Error = dbErr.String
		}
		j.StartDate, _ = time.Parse(dateFormat, startStr)
		j.EndDate, _ = time.Parse(dateFormat, endStr)
		j.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		j.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

func (r *Repository) FindActive(ctx context.Context, source, symbol string, from, to string) (*domain.Job, error) {
	const query = `SELECT id, source, symbol, start_date, end_date,
		status, error, records_count, created_at, updated_at
		FROM jobs
		WHERE source = ? AND symbol = ?
		  AND start_date = ? AND end_date = ?
		  AND status IN ('pending', 'running')
		LIMIT 1`

	j := &domain.Job{}
	var startStr, endStr, status, createdStr, updatedStr string
	var dbErr sql.NullString

	err := r.db.QueryRowContext(ctx, query, source, symbol, from, to).Scan(
		&j.ID, &j.Source, &j.Symbol,
		&startStr, &endStr, &status, &dbErr,
		&j.RecordsCount, &createdStr, &updatedStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find active job: %w", err)
	}

	j.Status = domain.Status(status)
	if dbErr.Valid {
		j.Error = dbErr.String
	}
	j.StartDate, _ = time.Parse(dateFormat, startStr)
	j.EndDate, _ = time.Parse(dateFormat, endStr)
	j.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	j.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return j, nil
}

func (r *Repository) ClaimPending(ctx context.Context) (*domain.Job, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("claim pending: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var id int64
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM jobs WHERE status = 'pending' ORDER BY id ASC LIMIT 1`,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim pending: select: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE jobs SET status = 'running', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ?`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("claim pending: update: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("claim pending: commit: %w", err)
	}

	return r.Get(ctx, id)
}

func (r *Repository) RecoverStale(ctx context.Context) (int64, error) {
	const query = `UPDATE jobs SET status = 'pending', error = NULL,
		updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE status = 'running'`

	res, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("recover stale jobs: %w", err)
	}

	return res.RowsAffected()
}
