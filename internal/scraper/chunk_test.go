package scraper

import (
	"testing"
	"time"
)

func date(m, d int) time.Time {
	y := 2024
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func TestSplitDateRange(t *testing.T) {
	tests := []struct {
		name      string
		from, to  time.Time
		chunkDays int
		wantLen   int
		wantFirst DateRange
		wantLast  DateRange
	}{
		{
			name:      "single chunk",
			from:      date(1, 1),
			to:        date(1, 10),
			chunkDays: 60,
			wantLen:   1,
			wantFirst: DateRange{From: date(1, 1), To: date(1, 10)},
			wantLast:  DateRange{From: date(1, 1), To: date(1, 10)},
		},
		{
			name:      "multiple chunks",
			from:      date(1, 1),
			to:        date(3, 31),
			chunkDays: 30,
			wantLen:   4,
			wantFirst: DateRange{From: date(1, 1), To: date(1, 30)},
			wantLast:  DateRange{From: date(3, 31), To: date(3, 31)},
		},
		{
			name:      "exact chunk boundary",
			from:      date(1, 1),
			to:        date(1, 30),
			chunkDays: 30,
			wantLen:   1,
			wantFirst: DateRange{From: date(1, 1), To: date(1, 30)},
			wantLast:  DateRange{From: date(1, 1), To: date(1, 30)},
		},
		{
			name:      "from after to returns nil",
			from:      date(3, 1),
			to:        date(1, 1),
			chunkDays: 30,
			wantLen:   0,
		},
		{
			name:      "zero chunk days returns nil",
			from:      date(1, 1),
			to:        date(1, 10),
			chunkDays: 0,
			wantLen:   0,
		},
		{
			name:      "same day",
			from:      date(1, 1),
			to:        date(1, 1),
			chunkDays: 30,
			wantLen:   1,
			wantFirst: DateRange{From: date(1, 1), To: date(1, 1)},
			wantLast:  DateRange{From: date(1, 1), To: date(1, 1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitDateRange(tt.from, tt.to, tt.chunkDays)
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
			if tt.wantLen == 0 {
				return
			}
			if got[0] != tt.wantFirst {
				t.Errorf("first = %v, want %v", got[0], tt.wantFirst)
			}
			if got[len(got)-1] != tt.wantLast {
				t.Errorf("last = %v, want %v", got[len(got)-1], tt.wantLast)
			}
		})
	}
}
