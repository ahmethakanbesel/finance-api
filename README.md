# Finance API

## Supported Data Sources

- [TEFAS](https://tefas.gov.tr/)
- [Yahoo Finance](https://finance.yahoo.com/)

## Usage

### Running the server

#### Download from releases

```bash
./finance-api serve
```

##### Assing to a different port

```bash
./finance-api serve --http=0.0.0.0:8080
```

#### Run from the source

```bash
go run main.go serve
```

#### Docker

```bash
docker compose up -d
```

### Routes

#### TEFAS

```
http://127.0.0.1:8090/api/v1/tefas/funds/:code
```

`:code` is the code of the fund to be fetched.

#### Yahoo Finance

```
http://127.0.0.1:8090/api/v1/yahoo/symbols/:symbol
```

`:symbol` is the symbol of the stock to be fetched. For example, for THYAO.IS
`:symbol` is `THYAO.IS`.

#### Admin UI

```
http://127.0.0.1:8090/_
```

Default email is `admin@example.com` and password is `1234567890`.

### Query parameters

- `startDate`: Start date of the data to be fetched. Format: `YYYY-MM-DD`
- `endDate`: Optional, if not provided `today` will be used. End date of the
  data to be fetched. Format: `YYYY-MM-DD`
- `currency`: Currency of the data to be fetched. Can be either `TRY` or `USD`
  - Note: Currently changing currency has no effect on the data.
- `format`: Optional. Format of the data to be fetched. Available formats:
  `json` or `csv`. Default is `json`.

```
http://127.0.0.1:8090/api/v1/yahoo/symbols/THYAO.IS?startDate=2023-06-01&endDate=2023-09-30&currency=TRY
```

### Use with Pandas

```python
import pandas as pd

API_URL = 'http://127.0.0.1:8090'

def get_data(symbol, start_date, end_date):
    url = f'{API_URL}/api/v1/yahoo/symbols/{symbol}?startDate={start_date}&endDate={end_date}&format=csv'
    df = pd.read_csv(url, parse_dates=['Date'])
    df.set_index('Date', inplace=True)
    return df
```

### Use as a Go package

Both `tefas` and `yahoo` packages can be used independently from the web API,
and they implement `Scraper` interface.

```golang
type Scraper interface {
	GetSymbolData(symbol string, startDate, endDate time.Time) (<-chan *SymbolPrice, error)
}
```

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
