package analyzer

import (
	"math"
	"testing"
	"time"

	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/domain"
)

func TestAnalyzeIntervalProducesFiniteValues(t *testing.T) {
	candles := make([]domain.Candle, 800)
	for i := range candles {
		trend := float64(i) * 0.03
		wave := math.Sin(float64(i)/11) * 3
		price := 100 + trend + wave
		candles[i] = domain.Candle{
			Symbol: "TESTUSDT", Interval: "1h",
			OpenTime: time.Unix(int64(i*3600), 0), CloseTime: time.Unix(int64((i+1)*3600-1), 0),
			Open: price - 0.2, High: price + 1, Low: price - 1, Close: price,
			Volume: 1000 + 100*math.Sin(float64(i)/7),
		}
	}
	engine := Engine{
		NearestNeighbors: 40, MinimumSamples: 120,
		MinimumProbability: 0.58, MinimumEdge: 0.06, SignalThreshold: 0.12,
		Lookahead: map[string]int{"1h": 12}, Targets: map[string]float64{"1h": 1.0}, Weights: map[string]float64{"1h": 1},
	}
	result, err := engine.AnalyzeInterval("TESTUSDT", "1h", candles)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []float64{result.BuyProbability, result.SellProbability, result.ExpectedReturn, result.Score, result.Confidence} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			t.Fatalf("invalid result value %v", value)
		}
	}
	if result.SampleCount < 120 || result.NearestCount != 40 {
		t.Fatalf("unexpected sample counts: total=%d nearest=%d", result.SampleCount, result.NearestCount)
	}
}
