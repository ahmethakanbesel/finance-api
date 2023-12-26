package utils

import (
	"github.com/pocketbase/pocketbase/models"
)

func PriceRecordstoCsv(records []*models.Record) (string, error) {
	var csvString string
	csvString += "Symbol,Date,Currency,Source,Close\n"
	for _, record := range records {
		csvString += record.GetString("symbol") + "," + record.GetDateTime("date").Time().Format("2006-01-02") + "," + record.GetString("currency") + "," + record.GetString("source") + "," + record.GetString("closePrice") + "\n"
	}
	return csvString, nil
}
