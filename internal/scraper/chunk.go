package scraper

import "time"

type DateRange struct {
	From time.Time
	To   time.Time
}

func SplitDateRange(from, to time.Time, chunkDays int) []DateRange {
	if from.After(to) || chunkDays <= 0 {
		return nil
	}

	var chunks []DateRange
	for cur := from; !cur.After(to); cur = cur.AddDate(0, 0, chunkDays) {
		end := cur.AddDate(0, 0, chunkDays-1)
		if end.After(to) {
			end = to
		}
		chunks = append(chunks, DateRange{From: cur, To: end})
	}
	return chunks
}
