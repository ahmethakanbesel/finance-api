package rate

import "time"

const PairUSDTRY = "USDTRY"

type Rate struct {
	ID        int64
	Pair      string
	Date      time.Time
	Rate      float64
	CreatedAt time.Time
}
