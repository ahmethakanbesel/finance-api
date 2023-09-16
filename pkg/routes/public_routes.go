package routes

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ahmethakanbesel/finance-api/app/scrapers"
	"github.com/ahmethakanbesel/finance-api/pkg/utils"
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

func setupRoute(app *pocketbase.PocketBase, endpoint string, scraperName string) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET(endpoint, func(c echo.Context) error {
			symbol := strings.ToUpper(c.PathParam("symbol"))
			if len(symbol) < 3 {
				return c.String(http.StatusBadRequest, "Symbol must be at least 3 characters")
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
				return c.String(http.StatusBadRequest, "Currency must be either TRY or USD")
			}

			record, _ := app.Dao().FindFirstRecordByFilter(
				"scrapes",
				"source = {:source} && symbol = {:symbol} && startDate = {:startDate} && endDate = {:endDate} && currency = {:currency}",
				dbx.Params{
					"source":    scraperName,
					"symbol":    symbol,
					"startDate": startDateStr + " 00:00:00.000Z",
					"endDate":   endDateStr + " 00:00:00.000Z",
					"currency":  currency,
				},
			)

			if record == nil {
				scraper, err := scrapers.CreateScraper(scraperName)
				if err != nil {
					return err
				}

				scrapedData, err := scraper.GetSymbolData(symbol, startDateStr, endDateStr)
				if err != nil {
					return err
				}

				collection, err := app.Dao().FindCollectionByNameOrId("prices")
				if err != nil {
					return err
				}

				for _, data := range scrapedData {
					if data.Date == "" || data.Close < 0 {
						continue
					}

					record := models.NewRecord(collection)
					record.Set("source", scraperName)
					record.Set("symbol", symbol)
					record.Set("date", data.Date)
					record.Set("closePrice", data.Close)
					record.Set("currency", currency)

					if err := app.Dao().SaveRecord(record); err != nil {
						fmt.Println(err)
					}
				}

				collection, err = app.Dao().FindCollectionByNameOrId("scrapes")
				if err != nil {
					return err
				}

				record := models.NewRecord(collection)
				record.Set("source", scraperName)
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
				"source = {:source} && symbol = {:symbol} && date >= {:startDate} && date <= {:endDate} && currency = {:currency}",
				"-date",
				0,
				0,
				dbx.Params{
					"source":    scraperName,
					"symbol":    symbol,
					"startDate": startDate,
					"endDate":   endDate,
					"currency":  currency,
				},
			)
			if err != nil {
				return err
			}

			format := c.QueryParam("format")

			if format == "csv" {
				csvString, err := utils.PriceRecordstoCsv(records)
				if err != nil {
					return err
				}
				return c.String(http.StatusOK, csvString)
			}

			return c.JSON(http.StatusOK, &GeneralResponse{
				Message: "ok",
				Data:    records,
			})
		}, apis.ActivityLogger(app))

		return nil
	})
}

func PublicRoutes(app *pocketbase.PocketBase) {
	setupRoute(app, "/api/yahoo/symbol/:symbol", "yahoo")
	setupRoute(app, "/api/tefas/fund/:symbol", "tefas")
}
