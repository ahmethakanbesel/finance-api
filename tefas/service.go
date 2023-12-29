package tefas

import (
	"database/sql"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/ahmethakanbesel/finance-api/scraper"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

const (
	serviceName = "tefas"
)

type Service interface {
	GetAndSaveFundData(fundCode, currency string, startDate, endDate time.Time) ([]*models.Record, error)
	GetScrape(fundCode, currenct string, startDate, endDate time.Time) (*models.Record, error)
}

type service struct {
	app     *pocketbase.PocketBase
	scraper scraper.Scraper
	wg      sync.WaitGroup
}

var _ Service = (*service)(nil)

func NewService(app *pocketbase.PocketBase, scraper scraper.Scraper) Service {
	return &service{
		app:     app,
		scraper: scraper,
	}
}

func (s *service) GetScrape(fundCode, currency string, startDate, endDate time.Time) (*models.Record, error) {
	record, err := s.app.Dao().FindFirstRecordByFilter(
		"scrapes",
		"source = {:source} && symbol = {:symbol} && startDate = {:startDate} && endDate = {:endDate} && currency = {:currency}",
		dbx.Params{
			"source":    serviceName,
			"symbol":    fundCode,
			"startDate": startDate.Format(dateFormat) + " 00:00:00.000Z",
			"endDate":   endDate.Format(dateFormat) + " 00:00:00.000Z",
			"currency":  currency,
		},
	)

	return record, err
}

func (s *service) GetAndSaveFundData(fundCode, currency string, startDate, endDate time.Time) ([]*models.Record, error) {
	dao := s.app.Dao()

	record, err := s.GetScrape(fundCode, currency, startDate, endDate)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if record == nil {
		dataChan, err := s.scraper.GetSymbolData(fundCode, startDate, endDate)
		if err != nil {
			return nil, err
		}

		collection, err := dao.FindCollectionByNameOrId("prices")
		if err != nil {
			return nil, err
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			for data := range dataChan {
				if data.Date == "" || data.Close < 0 {
					continue
				}

				record := models.NewRecord(collection)
				record.Set("source", serviceName)
				record.Set("symbol", fundCode)
				record.Set("date", data.Date)
				record.Set("closePrice", data.Close)
				record.Set("currency", currency)

				if err := dao.SaveRecord(record); err != nil {
					slog.Error("error saving scraped data", "error", err)
				}
			}
		}()

		s.wg.Wait()

		collection, err = dao.FindCollectionByNameOrId("scrapes")
		if err != nil {
			return nil, err
		}

		record := models.NewRecord(collection)
		record.Set("source", serviceName)
		record.Set("symbol", fundCode)
		record.Set("startDate", startDate)
		record.Set("endDate", endDate)
		record.Set("currency", "TRY")

		if err := dao.SaveRecord(record); err != nil {
			return nil, err
		}
	}

	records, err := dao.FindRecordsByFilter(
		"prices",
		"source = {:source} && symbol = {:symbol} && date >= {:startDate} && date <= {:endDate} && currency = {:currency}",
		"-date",
		0,
		0,
		dbx.Params{
			"source":    serviceName,
			"symbol":    fundCode,
			"startDate": startDate,
			"endDate":   endDate,
			"currency":  currency,
		},
	)
	if err != nil {
		return nil, err
	}

	return records, nil
}
