package marketdata

import (
	"time"
	"fmt"
	"sort"
)

type DateRange struct {
	From time.Time
	To   time.Time
}

func (d *DateRange) String() string {
	l := "2006-01-02 15:04:05"
	return fmt.Sprintf("From: %v To: %v", d.From.Format(l), d.To.Format(l))
}

type Tick struct {
	HasQuote bool
	HasTrade bool

	IsOpening bool
	IsClosing bool

	LastPrice float64
	LastSize  int64
	LastExch  string
	Datetime  time.Time

	BidExch string
	AskExch string

	BidPrice float64
	AskPrice float64
	BidSize  int64
	AskSize  int64

	CondQuote string
	Cond1     string
	Cond2     string
	Cond3     string
	Cond4     string
}

type Candle struct {
	Open         float64
	High         float64
	Low          float64
	Close        float64
	AdjClose     float64
	Volume       int64
	OpenInterest int64
	Datetime     time.Time
}

type QuoteSnapshot struct {
}

type CandleArray []*Candle
type TickArray []*Tick

func (t TickArray) Sort() TickArray {
	sort.SliceStable(t, func(i, j int) bool {
		return t[i].Datetime.Unix() > t[j].Datetime.Unix()
	})
	return t
}
