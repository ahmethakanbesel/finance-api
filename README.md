# Finance API

> **Disclaimer:** This project is for **educational purposes only** and is not intended for production financial use. Currency conversions use IEEE 754 floating-point arithmetic, which can introduce rounding errors. For real monetary calculations, use fixed-point or arbitrary-precision decimal libraries. Do not rely on this software for trading, accounting, or any financial decision-making.

A financial data aggregator API that scrapes and caches historical price data from multiple sources.

## Supported Data Sources

- [TEFAS](https://tefas.gov.tr/)
- [Yahoo Finance](https://finance.yahoo.com/)
- [IS Yatirim](https://www.isyatirim.com.tr/)

## Usage

### Running the Server

#### Download from Releases

```bash
./finance-api
```

#### Run from source

```bash
go run ./cmd/finance-api
```

#### Make

```bash
make run
```

#### Docker

```bash
docker run -p 8090:8090 -d --name finance-api -e PORT=8090 --restart unless-stopped finance-api
```

### Configuration

Environment variables with defaults:

| Variable  | Default      | Description          |
|-----------|--------------|----------------------|
| `PORT`    | `8080`       | HTTP server port     |
| `DB_PATH` | `finance.db` | SQLite database path |
| `WORKERS` | `5`          | Scraper concurrency  |

### API Routes

#### Health

```ascii
GET /health
```

#### List sources

```ascii
GET /api/v1/sources
```

#### Get prices

```ascii
GET /api/v1/prices/{symbol}
```

`{symbol}` is the fund code or ticker symbol (e.g. `YAC`, `AAPL`, `THYAO.IS`, `ALTINS1`).

**Query parameters:**

| Parameter   | Required | Default | Description                                       |
|-------------|----------|---------|---------------------------------------------------|
| `source`    | yes      |         | Data source: `tefas`, `yahoo`, or `isyatirim`     |
| `startDate` | yes      |         | Start date, format `YYYY-MM-DD`                   |
| `endDate`   | no       | today   | End date, format `YYYY-MM-DD`                     |
| `currency`  | no       | `TRY`   | `TRY` or `USD`                                    |
| `format`    | no       | `json`  | Response format: `json` or `csv`                  |

**Examples:**

```ascii
GET /api/v1/prices/YAC?source=tefas&startDate=2024-01-01&endDate=2024-01-31&currency=TRY
GET /api/v1/prices/AAPL?source=yahoo&startDate=2024-01-01&endDate=2024-01-31&currency=USD
GET /api/v1/prices/THYAO.IS?source=yahoo&startDate=2024-01-01&currency=TRY&format=csv
GET /api/v1/prices/ALTINS1?source=isyatirim&startDate=2025-01-01&endDate=2025-01-31&currency=TRY
GET /api/v1/prices/XAUUSD?source=isyatirim&startDate=2025-01-01&endDate=2025-01-31&currency=USD
GET /api/v1/prices/USDTRY?source=isyatirim&startDate=2025-01-01&endDate=2025-01-31&currency=TRY
```

#### Jobs

```ascii
GET /api/v1/jobs
GET /api/v1/jobs/{id}
```

Each scrape operation creates a tracked job. Use these endpoints to inspect job status and history.

### JSON Response Format

All JSON responses are wrapped in:

```json
{
  "message": "ok",
  "data": {
    "prices": [
      {
        "symbol": "YAC",
        "date": "2026-01-20T00:00:00Z",
        "closePrice": 0.367598,
        "currency": "USD",
        "nativePrice": 13.027306,
        "nativeCurrency": "TRY",
        "rate": 35.4379,
        "source": "tefas"
      },
      {
        "symbol": "YAC",
        "date": "2026-01-21T00:00:00Z",
        "closePrice": 0.366648,
        "currency": "USD",
        "nativePrice": 12.993608,
        "nativeCurrency": "TRY",
        "rate": 35.4379,
        "source": "tefas"
      }
    ]
  }
}
```

### Use with Pandas

```python
import pandas as pd

API_URL = "http://127.0.0.1:8080"

def get_data(source, symbol, start_date, end_date, currency="TRY"):
    url = f"{API_URL}/api/v1/prices/{symbol}?source={source}&startDate={start_date}&endDate={end_date}&currency={currency}&format=csv"
    df = pd.read_csv(url, parse_dates=["Date"])
    df.set_index("Date", inplace=True)
    return df
```

## Development

### Prerequisites

- Go 1.24+
- [golangci-lint](https://golangci-lint.run/) v2 (for linting)

### Commands

```bash
make build     # Build the binary
make run       # Build and run
make test      # Run tests with race detector
make lint      # Run golangci-lint
make check     # Run lint + format check
make fmt       # Format code
```
