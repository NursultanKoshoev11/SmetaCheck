package indicators

import (
	"math"
	"testing"
	"time"

	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/domain"
)

func TestComputeRisingMarket(t *testing.T) {
	candles := make([]domain.Candle, 100)
	for i := range candles {
		price := 100 + float64(i)
		candles[i] = domain.Candle{
			OpenTime:  time.Unix(int64(i*3600), 0),
			Open:      price - 0.3,
			High:      price + 1,
			Low:       price - 1,
			Close:     price,
			Volume:    100 + float64(i),
			CloseTime: time.Unix(int64((i+1)*3600-1), 0),
		}
	}
	features, err := Compute(candles, len(candles)-1)
	if err != nil {
		t.Fatal(err)
	}
	if features.RSI14 < 99 {
		t.Fatalf("expected high RSI, got %.2f", features.RSI14)
	}
	if features.EMAGap <= 0 || features.PriceEMA20 <= 0 {
		t.Fatalf("expected positive EMA trend, got gap %.3f price %.3f", features.EMAGap, features.PriceEMA20)
	}
	for _, value := range features.Vector() {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			t.Fatalf("invalid feature value %v", value)
		}
	}
}
