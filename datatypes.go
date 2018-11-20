package marketdata

import "time"

type Tick struct{
	HasQuote bool
	HasTrade bool

	LastPrice float64
	LastSize int64
	LastExch string
	Datetime time.Time

	BidExch string
	AskExch string

	BidPrice float64
	AskPrice float64
	BidSize int64
	AskSize int64

	CondQuote string
	Cond1 string
	Cond2 string
	Cond3 string
	Cond4 string

}



type Candle struct{
	//Symbol string
	Open float64
	High float64
	Low float64
	Close float64
	AdjClose float64
	Volume int64
	OpenInterest int64
	Datetime time.Time
}

type QuoteSnapshot struct{

}