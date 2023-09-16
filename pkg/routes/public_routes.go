package routes

import (
	"fmt"
	"strings"
	"time"

	"github.com/ahmethakanbesel/finance-api/app/scrapers"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

type GeneralResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func PublicRoutes(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/api/yahoo/symbol/:symbol", func(c echo.Context) error {
			symbol := strings.ToUpper(c.PathParam("symbol"))
			if len(symbol) < 3 {
				return fmt.Errorf("symbol must be at least 3 characters")
			}

			startDateStr := c.QueryParam("startDate")
			startDate, err := time.Parse("2006-01-02", startDateStr)
			if err != nil {
				return err
			}

			endDateStr := c.QueryParam("endDate")
			if endDateStr == "" {
				endDateStr = time.Now().Format("2006-01-02")
			}
			endDate, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				return err
			}

			currency := strings.ToUpper(c.QueryParam("currency"))
			if currency != "TRY" && currency != "USD" {
				return fmt.Errorf("currency must be either TRY or USD")
			}

			record, _ := app.Dao().FindFirstRecordByFilter(
				"scrapes",
				"source = 'yahoo' && symbol = {:symbol} && startDate = {:startDate} && endDate = {:endDate} && currency = {:currency}",
				dbx.Params{
					"symbol":    symbol,
					"startDate": startDateStr + " 00:00:00.000Z",
					"endDate":   endDateStr + " 00:00:00.000Z",
					"currency":  currency,
				},
			)

			if record == nil {
				scrapedData, err := scrapers.GetYahooSymbolData(symbol, startDateStr, endDateStr)
				if err != nil {
					return err
				}

				collection, err := app.Dao().FindCollectionByNameOrId("prices")
				if err != nil {
					return err
				}

				for _, data := range scrapedData {
					if data.Date == "" {
						continue
					}
					record := models.NewRecord(collection)
					record.Set("source", "yahoo")
					record.Set("symbol", symbol)
					record.Set("date", data.Date)
					record.Set("closePrice", data.Close)
					record.Set("currency", currency)

					if err := app.Dao().SaveRecord(record); err != nil {
						return err
					}
				}

				collection, err = app.Dao().FindCollectionByNameOrId("scrapes")
				if err != nil {
					return err
				}

				record := models.NewRecord(collection)
				record.Set("source", "yahoo")
				record.Set("symbol", symbol)
				record.Set("startDate", startDate)
				record.Set("endDate", endDate)
				record.Set("currency", "TRY")

				if err := app.Dao().SaveRecord(record); err != nil {
					return err
				}
			}

			records, err := app.Dao().FindRecordsByFilter(
				"prices",
				"source = 'yahoo' && symbol = {:symbol} && date >= {:startDate} && date <= {:endDate} && currency = {:currency}",
				"-date", // sort
				0,       // limit
				0,       // offset
				dbx.Params{
					"symbol":    symbol,
					"startDate": startDate,
					"endDate":   endDate,
					"currency":  currency,
				},
			)
			if err != nil {
				return err
			}

			return c.JSON(200, &GeneralResponse{
				Message: "ok",
				Data:    records,
			})
		}, apis.ActivityLogger(app))
		return nil
	})

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/api/tefas/fund/:code", func(c echo.Context) error {
			fundCode := strings.ToUpper(c.PathParam("code"))
			if len(fundCode) != 3 {
				return fmt.Errorf("fund code must be 3 characters")
			}

			startDateStr := c.QueryParam("startDate")
			startDate, err := time.Parse("2006-01-02", startDateStr)
			if err != nil {
				return err
			}

			endDateStr := c.QueryParam("endDate")
			if endDateStr == "" {
				endDateStr = time.Now().Format("2006-01-02")
			}
			endDate, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				return err
			}

			currency := strings.ToUpper(c.QueryParam("currency"))
			if currency != "TRY" && currency != "USD" {
				return fmt.Errorf("currency must be either TRY or USD")
			}

			record, _ := app.Dao().FindFirstRecordByFilter(
				"scrapes",
				"source = 'tefas' && symbol = {:symbol} && startDate = {:startDate} && endDate = {:endDate} && currency = {:currency}",
				dbx.Params{
					"symbol":    fundCode,
					"startDate": startDateStr + " 00:00:00.000Z",
					"endDate":   endDateStr + " 00:00:00.000Z",
					"currency":  currency,
				},
			)

			if record == nil {
				scrapedData, err := scrapers.GetTefasFundData(fundCode, startDateStr, endDateStr)
				if err != nil {
					return err
				}

				collection, err := app.Dao().FindCollectionByNameOrId("prices")
				if err != nil {
					return err
				}

				for _, data := range scrapedData {
					record := models.NewRecord(collection)
					record.Set("source", "tefas")
					record.Set("symbol", fundCode)
					record.Set("date", data.Date)
					record.Set("closePrice", data.Price)
					record.Set("currency", "TRY")

					if err := app.Dao().SaveRecord(record); err != nil {
						return err
					}
				}

				collection, err = app.Dao().FindCollectionByNameOrId("scrapes")
				if err != nil {
					return err
				}

				record := models.NewRecord(collection)
				record.Set("source", "tefas")
				record.Set("symbol", fundCode)
				record.Set("startDate", startDate)
				record.Set("endDate", endDate)
				record.Set("currency", "TRY")

				if err := app.Dao().SaveRecord(record); err != nil {
					return err
				}
			}

			records, err := app.Dao().FindRecordsByFilter(
				"prices",
				"source = 'tefas' && symbol = {:symbol} && date >= {:startDate} && date <= {:endDate} && currency = {:currency}",
				"-date", // sort
				0,       // limit
				0,       // offset
				dbx.Params{
					"symbol":    fundCode,
					"startDate": startDate,
					"endDate":   endDate,
					"currency":  currency,
				},
			)
			if err != nil {
				return err
			}

			return c.JSON(200, &GeneralResponse{
				Message: "ok",
				Data:    records,
			})
		}, apis.ActivityLogger(app))
		return nil
	})
}
