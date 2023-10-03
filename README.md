# Finance API

## Supported Data Sources

- [TEFAS](https://tefas.gov.tr/)
- [Yahoo Finance](https://finance.yahoo.com/)

## Usage

### Running the Server

#### Download from Releases

```bash
./finance-api serve
```

##### Assing to a different port

```bash
./finance-api serve --http=0.0.0.0:8080
```

#### Run from Source

```bash
go run main.go serve
```

#### Docker

```bash
docker run -p 8090:8090 -d --name finance-api -e PORT=8090 --restart unless-stopped finance-api
```

### Routes

#### TEFAS

```
http://127.0.0.1:8090/api/tefas/fund/:code
```

`:code` is the code of the fund to be fetched.

#### Yahoo Finance

```
http://127.0.0.1:8090/api/yahoo/symbol/:symbol
```

`:symbol` is the symbol of the stock to be fetched. For example, for THYAO.IS
`:symbol` is `THYAO.IS`.

#### Admin UI

```
http://127.0.0.1:8090/_
```

Default email is `admin@example.com` and password is `1234567890`.

### Query Parameters

- `startDate`: Start date of the data to be fetched. Format: `YYYY-MM-DD`
- `endDate`: Optional, if not provided `today` will be used. End date of the
  data to be fetched. Format: `YYYY-MM-DD`
- `currency`: Currency of the data to be fetched. Can be either `TRY` or `USD`
  - Note: Currently changing currency has no effect on the data.
- `format`: Optional. Format of the data to be fetched. Available formats:
  `json` or `csv`. Default is `json`.

```
http://127.0.0.1:8090/api/yahoo/symbol/THYAO.IS?startDate=2023-06-01&endDate=2023-09-30&currency=TRY
```

### Use with Pandas

```python
import pandas as pd

API_URL = 'http://127.0.0.1:8090'

def get_data(symbol, start_date, end_date):
    url = f'{API_URL}/api/yahoo/symbol/{symbol}?startDate={start_date}&endDate={end_date}&format=csv'
    df = pd.read_csv(url, parse_dates=['Date'])
    df.set_index('Date', inplace=True)
    return df
```

## Demo

- `TEFAS (json)`

```
https://finans.dokuz.gen.tr/api/tefas/fund/HKP?startDate=2023-06-01&endDate=2023-09-30&currency=TRY
```

- `Yahoo Finance (csv)`

```
https://finans.dokuz.gen.tr/api/yahoo/symbol/THYAO.IS?startDate=2023-06-01&endDate=2023-09-30&currency=TRY&format=csv
```

## Credits

- [Pocketbase](https://github.com/pocketbase/pocketbase)
