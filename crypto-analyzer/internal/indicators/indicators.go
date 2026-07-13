package indicators

import (
	"errors"
	"math"

	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/domain"
)

const minimumIndex = 55

func MinimumIndex() int { return minimumIndex }

func Compute(candles []domain.Candle, index int) (domain.Features, error) {
	if index < minimumIndex || index >= len(candles) {
		return domain.Features{}, errors.New("not enough candles for indicators")
	}
	closePrice := candles[index].Close
	if closePrice <= 0 {
		return domain.Features{}, errors.New("invalid close price")
	}

	ema20 := emaAt(candles, index, 20)
	ema50 := emaAt(candles, index, 50)
	atr := atrAt(candles, index, 14)
	rsi := rsiAt(candles, index, 14)
	mean20, std20 := closeMeanStd(candles, index, 20)
	volumeMean := meanVolume(candles, index-1, 20)
	current := candles[index]
	rangeSize := current.High - current.Low
	if rangeSize <= 0 {
		rangeSize = closePrice * 1e-9
	}
	bodyTop := math.Max(current.Open, current.Close)
	bodyBottom := math.Min(current.Open, current.Close)
	upperWick := math.Max(0, current.High-bodyTop)
	lowerWick := math.Max(0, bodyBottom-current.Low)
	breakout := breakoutAt(candles, index, 20)
	bbLower := mean20 - 2*std20
	bbUpper := mean20 + 2*std20
	bbPosition := 0.5
	if bbUpper > bbLower {
		bbPosition = (closePrice - bbLower) / (bbUpper - bbLower)
	}
	volumeRatio := 1.0
	if volumeMean > 0 {
		volumeRatio = current.Volume / volumeMean
	}
	bbWidth := 0.0
	if mean20 != 0 {
		bbWidth = (bbUpper - bbLower) / mean20 * 100
	}

	return domain.Features{
		Return1:      pctChange(candles[index-1].Close, closePrice),
		Return3:      pctChange(candles[index-3].Close, closePrice),
		Return6:      pctChange(candles[index-6].Close, closePrice),
		RSI14:        rsi,
		EMAGap:       pctChange(ema50, ema20),
		PriceEMA20:   pctChange(ema20, closePrice),
		ATR14Pct:     atr / closePrice * 100,
		VolumeRatio:  volumeRatio,
		BBPosition:   bbPosition,
		BBWidthPct:   bbWidth,
		BodyPct:      math.Abs(current.Close-current.Open) / rangeSize,
		UpperWickPct: upperWick / rangeSize,
		LowerWickPct: lowerWick / rangeSize,
		Breakout20:   breakout,
	}, nil
}

func pctChange(from, to float64) float64 {
	if from == 0 {
		return 0
	}
	return (to - from) / from * 100
}

func emaAt(candles []domain.Candle, index, period int) float64 {
	start := index - period*3
	if start < 0 {
		start = 0
	}
	ema := candles[start].Close
	alpha := 2.0 / float64(period+1)
	for i := start + 1; i <= index; i++ {
		ema = alpha*candles[i].Close + (1-alpha)*ema
	}
	return ema
}

func rsiAt(candles []domain.Candle, index, period int) float64 {
	start := index - period
	if start < 1 {
		return 50
	}
	gain, loss := 0.0, 0.0
	for i := start + 1; i <= index; i++ {
		delta := candles[i].Close - candles[i-1].Close
		if delta >= 0 {
			gain += delta
		} else {
			loss -= delta
		}
	}
	if loss == 0 {
		if gain == 0 {
			return 50
		}
		return 100
	}
	rs := (gain / float64(period)) / (loss / float64(period))
	return 100 - 100/(1+rs)
}

func atrAt(candles []domain.Candle, index, period int) float64 {
	start := index - period + 1
	if start < 1 {
		start = 1
	}
	total := 0.0
	count := 0
	for i := start; i <= index; i++ {
		tr := math.Max(candles[i].High-candles[i].Low,
			math.Max(math.Abs(candles[i].High-candles[i-1].Close), math.Abs(candles[i].Low-candles[i-1].Close)))
		total += tr
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func closeMeanStd(candles []domain.Candle, index, period int) (float64, float64) {
	start := index - period + 1
	if start < 0 {
		start = 0
	}
	mean := 0.0
	count := 0
	for i := start; i <= index; i++ {
		mean += candles[i].Close
		count++
	}
	mean /= float64(count)
	variance := 0.0
	for i := start; i <= index; i++ {
		d := candles[i].Close - mean
		variance += d * d
	}
	variance /= float64(count)
	return mean, math.Sqrt(variance)
}

func meanVolume(candles []domain.Candle, end, period int) float64 {
	if end < 0 {
		return 0
	}
	start := end - period + 1
	if start < 0 {
		start = 0
	}
	total := 0.0
	count := 0
	for i := start; i <= end; i++ {
		total += candles[i].Volume
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func breakoutAt(candles []domain.Candle, index, period int) float64 {
	start := index - period
	if start < 0 {
		start = 0
	}
	highest := candles[start].High
	lowest := candles[start].Low
	for i := start + 1; i < index; i++ {
		if candles[i].High > highest {
			highest = candles[i].High
		}
		if candles[i].Low < lowest {
			lowest = candles[i].Low
		}
	}
	if candles[index].Close > highest {
		return 1
	}
	if candles[index].Close < lowest {
		return -1
	}
	return 0
}
