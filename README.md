# Finance API

> **Disclaimer:** This project is for **educational purposes only** and is not intended for production financial use. Currency conversions use IEEE 754 floating-point arithmetic, which can introduce rounding errors. For real monetary calculations, use fixed-point or arbitrary-precision decimal libraries. Do not rely on this software for trading, accounting, or any financial decision-making.

A financial data aggregator API that scrapes and caches historical price data from multiple sources.

## Supported Data Sources

- [TEFAS](https://tefas.gov.tr/)
- [Yahoo Finance](https://finance.yahoo.com/)
- [IS Yatirim](https://www.isyatirim.com.tr/)

## Usage

### Running the server

#### Download from releases

```bash
./finance-api
```

#### Run from source

```bash
go run ./cmd/finance-api
```

<<<<<<< HEAD
#### Run from the source
=======
#### Make
>>>>>>> ab37e3d (docs: update readme)

```bash
make run
```

#### Docker

```bash
docker compose up -d
```

### Configuration

Environment variables with defaults:

<<<<<<< HEAD
```
http://127.0.0.1:8090/api/v1/tefas/funds/:code
=======
| Variable  | Default      | Description          |
|-----------|--------------|----------------------|
| `PORT`    | `8080`       | HTTP server port     |
| `DB_PATH` | `finance.db` | SQLite database path |
| `WORKERS` | `5`          | Scraper concurrency  |

### API Routes

#### Health

```ascii
GET /health
>>>>>>> ab37e3d (docs: update readme)
```

#### List sources

<<<<<<< HEAD
#### Yahoo Finance

```
http://127.0.0.1:8090/api/v1/yahoo/symbols/:symbol
=======
```ascii
GET /api/v1/sources
>>>>>>> ab37e3d (docs: update readme)
```

#### Get prices

```ascii
GET /api/v1/prices/{symbol}
```

`{symbol}` is the fund code or ticker symbol (e.g. `YAC`, `AAPL`, `THYAO.IS`, `ALTINS1`).

<<<<<<< HEAD
### Query parameters
=======
**Query parameters:**
>>>>>>> ab37e3d (docs: update readme)

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
<<<<<<< HEAD

```golang
package main

import (
	"fmt"
	"time"

	"github.com/ahmethakanbesel/finance-api/tefas"
	"github.com/ahmethakanbesel/finance-api/yahoo"
)

func main() {
	ctx := context.Background()

	tefasScraper := tefas.NewScraper(
		tefas.WithWorkers(5),
	)

	// get last year's data for the given fund
	tefasData, err := tefasScraper.GetSymbolData(ctx, "FUNDCODE", time.Now().AddDate(-1, 0, 0), time.Now())
	if err != nil {
		// handle error
	}

	for data := range tefasData {
		fmt.Println(data.Date, data.Close)
	}

	yahooScraper := yahoo.NewScraper(
		yahoo.WithWorkers(5),
	)

	// get last year's data for the given symbol
	yahooData, err := yahooScraper.GetSymbolData(ctx, "SYMBOLCODE", time.Now().AddDate(-1, 0, 0), time.Now())
	if err != nil {
		// handle error
	}

	for data := range yahooData {
		fmt.Println(data.Date, data.Close)
	}
}
```

## Demo

- `TEFAS (json)`

```
https://finans.dokuz.gen.tr/api/v1/tefas/funds/HKP?startDate=2023-06-01&endDate=2023-09-30&currency=TRY
```

- `Yahoo Finance (csv)`

```
https://finans.dokuz.gen.tr/api/v1/yahoo/symbols/THYAO.IS?startDate=2023-06-01&endDate=2023-09-30&currency=TRY&format=csv
```

## Web UI

The web UI is incomplete, and it is not ready to work out of the box. The source
code can be found under `/ui` folder.

### Preview

![web ui preview](/docs/web-ui-preview.png "web ui preview")

## Credits

- [Pocketbase](https://github.com/pocketbase/pocketbase)
- [Tremor](https://www.tremor.so/)
=======
>>>>>>> ab37e3d (docs: update readme)
