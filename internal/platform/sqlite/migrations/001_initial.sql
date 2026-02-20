CREATE TABLE IF NOT EXISTS prices (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source      TEXT NOT NULL,
    symbol      TEXT NOT NULL,
    date        TEXT NOT NULL,
    close_price REAL NOT NULL,
    currency    TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(source, symbol, date)
);
CREATE INDEX IF NOT EXISTS idx_prices_lookup ON prices (source, symbol, date);

CREATE TABLE IF NOT EXISTS jobs (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    source         TEXT    NOT NULL,
    symbol         TEXT    NOT NULL,
    start_date     TEXT    NOT NULL,
    end_date       TEXT    NOT NULL,
    status         TEXT    NOT NULL DEFAULT 'pending',
    error          TEXT,
    records_count  INTEGER NOT NULL DEFAULT 0,
    created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_jobs_lookup ON jobs (source, symbol, status);

CREATE TABLE IF NOT EXISTS exchange_rates (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    pair       TEXT NOT NULL,
    date       TEXT NOT NULL,
    rate       REAL NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(pair, date)
);
CREATE INDEX IF NOT EXISTS idx_rates_lookup ON exchange_rates (pair, date);
