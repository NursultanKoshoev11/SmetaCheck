package domain

import "time"

type Symbol struct {
	Symbol      string  `json:"symbol"`
	BaseAsset   string  `json:"base_asset"`
	QuoteAsset  string  `json:"quote_asset"`
	QuoteVolume float64 `json:"quote_volume_24h"`
}

type Candle struct {
	Symbol         string    `json:"symbol"`
	Interval       string    `json:"interval"`
	OpenTime       time.Time `json:"open_time"`
	CloseTime      time.Time `json:"close_time"`
	Open           float64   `json:"open"`
	High           float64   `json:"high"`
	Low            float64   `json:"low"`
	Close          float64   `json:"close"`
	Volume         float64   `json:"volume"`
	QuoteVolume    float64   `json:"quote_volume"`
	Trades         int64     `json:"trades"`
	TakerBuyVolume float64   `json:"taker_buy_volume"`
}

type Features struct {
	Return1      float64 `json:"return_1"`
	Return3      float64 `json:"return_3"`
	Return6      float64 `json:"return_6"`
	RSI14        float64 `json:"rsi_14"`
	EMAGap       float64 `json:"ema_gap_pct"`
	PriceEMA20   float64 `json:"price_ema20_pct"`
	ATR14Pct     float64 `json:"atr_14_pct"`
	VolumeRatio  float64 `json:"volume_ratio_20"`
	BBPosition   float64 `json:"bb_position"`
	BBWidthPct   float64 `json:"bb_width_pct"`
	BodyPct      float64 `json:"body_pct"`
	UpperWickPct float64 `json:"upper_wick_pct"`
	LowerWickPct float64 `json:"lower_wick_pct"`
	Breakout20   float64 `json:"breakout_20"`
}

func (f Features) Vector() []float64 {
	return []float64{
		f.Return1, f.Return3, f.Return6, f.RSI14, f.EMAGap,
		f.PriceEMA20, f.ATR14Pct, f.VolumeRatio, f.BBPosition,
		f.BBWidthPct, f.BodyPct, f.UpperWickPct, f.LowerWickPct,
		f.Breakout20,
	}
}

type IntervalAnalysis struct {
	Symbol          string    `json:"symbol"`
	Interval        string    `json:"interval"`
	Price           float64   `json:"price"`
	Action          string    `json:"action"`
	BuyProbability  float64   `json:"buy_probability"`
	SellProbability float64   `json:"sell_probability"`
	BuyBaseline     float64   `json:"buy_baseline"`
	SellBaseline    float64   `json:"sell_baseline"`
	ExpectedReturn  float64   `json:"expected_return_pct"`
	SampleCount     int       `json:"sample_count"`
	NearestCount    int       `json:"nearest_count"`
	Distance        float64   `json:"average_distance"`
	Score           float64   `json:"score"`
	Confidence      float64   `json:"confidence_score"`
	LookaheadBars   int       `json:"lookahead_bars"`
	TargetPct       float64   `json:"target_pct"`
	Features        Features  `json:"features"`
	Reasons         []string  `json:"reasons"`
	CandleOpenTime  time.Time `json:"candle_open_time"`
}

type FuturesMetrics struct {
	Available       bool      `json:"available"`
	FundingRate     float64   `json:"funding_rate"`
	OpenInterest    float64   `json:"open_interest"`
	OpenInterestPct float64   `json:"open_interest_change_pct"`
	FetchedAt       time.Time `json:"fetched_at"`
}

type Signal struct {
	ID              int64              `json:"id"`
	ScanRunID       int64              `json:"scan_run_id"`
	Symbol          string             `json:"symbol"`
	Action          string             `json:"action"`
	Price           float64            `json:"price"`
	Score           float64            `json:"score"`
	Confidence      float64            `json:"confidence_score"`
	ExpectedReturn  float64            `json:"expected_return_pct"`
	SampleCount     int                `json:"sample_count"`
	FundingRate     float64            `json:"funding_rate"`
	OpenInterest    float64            `json:"open_interest"`
	OpenInterestPct float64            `json:"open_interest_change_pct"`
	Reasons         []string           `json:"reasons"`
	Intervals       []IntervalAnalysis `json:"intervals,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
}

type ScanRun struct {
	ID              int64     `json:"id"`
	StartedAt       time.Time `json:"started_at"`
	FinishedAt      time.Time `json:"finished_at"`
	Status          string    `json:"status"`
	SymbolsSelected int       `json:"symbols_selected"`
	SymbolsAnalyzed int       `json:"symbols_analyzed"`
	SignalsCreated  int       `json:"signals_created"`
	ErrorMessage    string    `json:"error_message"`
}
