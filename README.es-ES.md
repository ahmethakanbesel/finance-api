

# API Financiera

> **Aviso:** Este proyecto es **únicamente con fines educativos** y no está diseñado para uso financiero en producción. Las conversiones de moneda utilizan aritmética de punto flotante IEEE 754, lo cual puede introducir errores de redondeo. Para cálculos monetarios reales, utilice bibliotecas de punto fijo o decimales de precisión arbitraria. No confíe en este software para trading, contabilidad o cualquier toma de decisiones financieras.

Una API agregadora de datos financieros que extrae y almacena en caché datos históricos de precios de múltiples fuentes.

## Fuentes de datos compatibles

- [TEFAS](https://tefas.gov.tr/)
- [Yahoo Finance](https://finance.yahoo.com/)
- [IS Yatirim](https://www.isyatirim.com.tr/)

## Uso

### Ejecutar el servidor

#### Descargar desde los lanzamientos

```bash
./finance-api
```

#### Ejecutar desde el código fuente

```bash
go run ./cmd/finance-api
```

#### Make

```bash
make run
```

#### Docker

```bash
docker compose up -d
```

### Configuración

Variables de entorno con valores predeterminados:

| Variable  | Valor predeterminado | Descripción                     |
|-----------|----------------------|---------------------------------|
| `PORT`    | `8080`               | Puerto del servidor HTTP        |
| `DB_PATH` | `finance.db`         | Ruta de la base de datos SQLite |
| `WORKERS` | `5`                  | Concurrencia del scraper        |

### Rutas de la API

#### Estado

```ascii
GET /health
```

#### Listar fuentes

```ascii
GET /api/v1/sources
```

#### Obtener precios

```ascii
GET /api/v1/prices/{symbol}
```

`{symbol}` es el código del fondo o símbolo de cotización (p. ej., `YAC`, `AAPL`, `THYAO.IS`, `ALTINS1`).

**Parámetros de consulta:**

| Parámetro   | Requerido | Valor predeterminado | Descripción                                            |
|-------------|-----------|----------------------|--------------------------------------------------------|
| `source`    | sí        |                      | Fuente de datos: `tefas`, `yahoo` o `isyatirim`        |
| `startDate` | sí        |                      | Fecha de inicio, formato `YYYY-MM-DD`                  |
| `endDate`   | no        | hoy                  | Fecha de fin, formato `YYYY-MM-DD`                     |
| `currency`  | no        | `TRY`                | `TRY` o `USD`                                          |
| `format`    | no        | `json`               | Formato de respuesta: `json` o `csv`                   |

**Ejemplos:**

```ascii
GET /api/v1/prices/YAC?source=tefas&startDate=2024-01-01&endDate=2024-01-31&currency=TRY
GET /api/v1/prices/AAPL?source=yahoo&startDate=2024-01-01&endDate=2024-01-31&currency=USD
GET /api/v1/prices/THYAO.IS?source=yahoo&startDate=2024-01-01&currency=TRY&format=csv
GET /api/v1/prices/ALTINS1?source=isyatirim&startDate=2025-01-01&endDate=2025-01-31&currency=TRY
GET /api/v1/prices/GC=F?source=yahoo&startDate=2025-01-01&endDate=2025-01-31&currency=USD
GET /api/v1/prices/USDTRY=X?source=yahoo&startDate=2025-01-01&endDate=2025-01-31&currency=TRY
```

#### Trabajos

```ascii
GET /api/v1/jobs
GET /api/v1/jobs/{id}
```

Cada operación de extracción crea un trabajo rastreado. Utilice estos puntos de extremo para inspeccionar el estado y el historial de los trabajos.

### Formato de respuesta JSON

Todas las respuestas JSON se envuelven en:

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

### Uso con Pandas

```python
import pandas as pd

API_URL = "http://127.0.0.1:8080"

def get_data(source, symbol, start_date, end_date, currency="TRY"):
    url = f"{API_URL}/api/v1/prices/{symbol}?source={source}&startDate={start_date}&endDate={end_date}&currency={currency}&format=csv"
    df = pd.read_csv(url, parse_dates=["Date"])
    df.set_index("Date", inplace=True)
    return df
```

## Desarrollo

### Requisitos previos

- Go 1.24+
- [golangci-lint](https://golangci-lint.run/) v2 (para análisis de código)

### Comandos

```bash
make build     # Build the binary
make run       # Build and run
make test      # Run tests with race detector
make lint      # Run golangci-lint
make check     # Run lint + format check
make fmt       # Format code
```
