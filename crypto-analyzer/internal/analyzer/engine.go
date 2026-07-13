package analyzer

import (
	"fmt"
	"math"
	"sort"

	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/domain"
	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/indicators"
)

type Engine struct {
	NearestNeighbors   int
	MinimumSamples     int
	MinimumProbability float64
	MinimumEdge        float64
	SignalThreshold    float64
	Lookahead          map[string]int
	Targets            map[string]float64
	Weights            map[string]float64
}

type historicalSample struct {
	features      domain.Features
	futureReturn  float64
	hitBuyTarget  bool
	hitSellTarget bool
	distance      float64
}

func (e Engine) AnalyzeInterval(symbol, interval string, candles []domain.Candle) (domain.IntervalAnalysis, error) {
	lookahead, ok := e.Lookahead[interval]
	if !ok {
		return domain.IntervalAnalysis{}, fmt.Errorf("unsupported interval %s", interval)
	}
	target := e.Targets[interval]
	lastIndex := len(candles) - 1
	if lastIndex < indicators.MinimumIndex()+lookahead+e.MinimumSamples {
		return domain.IntervalAnalysis{}, fmt.Errorf("not enough history: have %d candles", len(candles))
	}
	current, err := indicators.Compute(candles, lastIndex)
	if err != nil {
		return domain.IntervalAnalysis{}, err
	}

	samples := make([]historicalSample, 0, lastIndex-indicators.MinimumIndex()-lookahead)
	for i := indicators.MinimumIndex(); i+lookahead < lastIndex; i++ {
		features, featureErr := indicators.Compute(candles, i)
		if featureErr != nil {
			continue
		}
		futureReturn := pctChange(candles[i].Close, candles[i+lookahead].Close)
		maxReturn, minReturn := futureExtremes(candles, i, lookahead)
		samples = append(samples, historicalSample{
			features: features, futureReturn: futureReturn,
			hitBuyTarget: maxReturn >= target, hitSellTarget: minReturn <= -target,
		})
	}
	if len(samples) < e.MinimumSamples {
		return domain.IntervalAnalysis{}, fmt.Errorf("only %d historical samples, minimum %d", len(samples), e.MinimumSamples)
	}

	means, stds := featureStats(samples)
	for i := range samples {
		samples[i].distance = normalizedDistance(current.Vector(), samples[i].features.Vector(), means, stds)
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].distance < samples[j].distance })
	neighborCount := e.NearestNeighbors
	if neighborCount > len(samples) {
		neighborCount = len(samples)
	}
	neighbors := samples[:neighborCount]

	buyCount, sellCount := 0, 0
	buyBaselineCount, sellBaselineCount := 0, 0
	expected := 0.0
	distance := 0.0
	for _, sample := range samples {
		if sample.hitBuyTarget {
			buyBaselineCount++
		}
		if sample.hitSellTarget {
			sellBaselineCount++
		}
	}
	for _, sample := range neighbors {
		if sample.hitBuyTarget {
			buyCount++
		}
		if sample.hitSellTarget {
			sellCount++
		}
		expected += sample.futureReturn
		distance += sample.distance
	}

	// Laplace smoothing prevents tiny samples from producing 0% or 100% claims.
	buyProbability := float64(buyCount+1) / float64(neighborCount+2)
	sellProbability := float64(sellCount+1) / float64(neighborCount+2)
	buyBaseline := float64(buyBaselineCount+1) / float64(len(samples)+2)
	sellBaseline := float64(sellBaselineCount+1) / float64(len(samples)+2)
	expected /= float64(neighborCount)
	distance /= float64(neighborCount)
	score := buyProbability - sellProbability
	action := "HOLD"
	selectedProbability := math.Max(buyProbability, sellProbability)
	selectedBaseline := buyBaseline
	if sellProbability > buyProbability {
		selectedBaseline = sellBaseline
	}
	edge := selectedProbability - selectedBaseline
	if buyProbability >= e.MinimumProbability && buyProbability-buyBaseline >= e.MinimumEdge && expected > 0 {
		action = "BUY"
	}
	if sellProbability >= e.MinimumProbability && sellProbability-sellBaseline >= e.MinimumEdge && expected < 0 {
		action = "SELL"
	}
	confidence := clamp((selectedProbability*0.70+clamp(edge*3, 0, 1)*0.20+clamp(float64(neighborCount)/float64(e.NearestNeighbors), 0, 1)*0.10)*100, 0, 100)

	analysis := domain.IntervalAnalysis{
		Symbol:          symbol,
		Interval:        interval,
		Price:           candles[lastIndex].Close,
		Action:          action,
		BuyProbability:  buyProbability,
		SellProbability: sellProbability,
		BuyBaseline:     buyBaseline,
		SellBaseline:    sellBaseline,
		ExpectedReturn:  expected,
		SampleCount:     len(samples),
		NearestCount:    neighborCount,
		Distance:        distance,
		Score:           score,
		Confidence:      confidence,
		LookaheadBars:   lookahead,
		TargetPct:       target,
		Features:        current,
		CandleOpenTime:  candles[lastIndex].OpenTime,
	}
	analysis.Reasons = intervalReasons(analysis)
	return analysis, nil
}

func (e Engine) Aggregate(symbol string, analyses []domain.IntervalAnalysis) domain.Signal {
	var weightedScore, totalWeight, expected float64
	var price float64
	samples := 0
	buyVotes, sellVotes := 0, 0
	for _, item := range analyses {
		weight := e.Weights[item.Interval]
		if weight <= 0 {
			weight = 1
		}
		weightedScore += item.Score * weight
		expected += item.ExpectedReturn * weight
		totalWeight += weight
		price = item.Price
		samples += item.SampleCount
		if item.Action == "BUY" {
			buyVotes++
		}
		if item.Action == "SELL" {
			sellVotes++
		}
	}
	if totalWeight > 0 {
		weightedScore /= totalWeight
		expected /= totalWeight
	}
	action := "HOLD"
	if weightedScore >= e.SignalThreshold && buyVotes > sellVotes {
		action = "BUY"
	}
	if weightedScore <= -e.SignalThreshold && sellVotes > buyVotes {
		action = "SELL"
	}
	agreement := 0.0
	if len(analyses) > 0 {
		agreement = float64(max(buyVotes, sellVotes)) / float64(len(analyses))
	}
	confidence := clamp(45+math.Abs(weightedScore)*180+agreement*15, 0, 100)

	signal := domain.Signal{
		Symbol:         symbol,
		Action:         action,
		Price:          price,
		Score:          weightedScore,
		Confidence:     confidence,
		ExpectedReturn: expected,
		SampleCount:    samples,
		Intervals:      analyses,
	}
	signal.Reasons = aggregateReasons(signal)
	return signal
}

func ApplyFuturesMetrics(signal *domain.Signal, metrics domain.FuturesMetrics) {
	if signal == nil || !metrics.Available {
		return
	}
	signal.FundingRate = metrics.FundingRate
	signal.OpenInterest = metrics.OpenInterest
	signal.OpenInterestPct = metrics.OpenInterestPct
	if math.Abs(metrics.FundingRate) >= 0.001 {
		direction := "положительный"
		if metrics.FundingRate < 0 {
			direction = "отрицательный"
		}
		signal.Reasons = append(signal.Reasons, fmt.Sprintf("Funding %s: %.4f%%", direction, metrics.FundingRate*100))
	}
	if math.Abs(metrics.OpenInterestPct) >= 1 {
		signal.Reasons = append(signal.Reasons, fmt.Sprintf("Open interest за последний час: %+.2f%%", metrics.OpenInterestPct))
	}
}

func featureStats(samples []historicalSample) ([]float64, []float64) {
	dimensions := len(samples[0].features.Vector())
	means := make([]float64, dimensions)
	stds := make([]float64, dimensions)
	for _, sample := range samples {
		vector := sample.features.Vector()
		for i, value := range vector {
			means[i] += value
		}
	}
	for i := range means {
		means[i] /= float64(len(samples))
	}
	for _, sample := range samples {
		vector := sample.features.Vector()
		for i, value := range vector {
			d := value - means[i]
			stds[i] += d * d
		}
	}
	for i := range stds {
		stds[i] = math.Sqrt(stds[i] / float64(len(samples)))
		if stds[i] < 1e-9 {
			stds[i] = 1
		}
	}
	return means, stds
}

func normalizedDistance(current, historical, means, stds []float64) float64 {
	weights := []float64{1.2, 1.1, 1.0, 0.8, 1.2, 1.1, 1.0, 1.2, 0.8, 0.9, 0.5, 0.5, 0.5, 1.0}
	total, weightTotal := 0.0, 0.0
	for i := range current {
		currentZ := clamp((current[i]-means[i])/stds[i], -5, 5)
		historyZ := clamp((historical[i]-means[i])/stds[i], -5, 5)
		d := currentZ - historyZ
		weight := 1.0
		if i < len(weights) {
			weight = weights[i]
		}
		total += weight * d * d
		weightTotal += weight
	}
	return math.Sqrt(total / weightTotal)
}

func intervalReasons(item domain.IntervalAnalysis) []string {
	reasons := []string{
		fmt.Sprintf("Похожие случаи: %d из %d исторических точек", item.NearestCount, item.SampleCount),
		fmt.Sprintf("Рост ≥ %.1f%%: %.1f%% (база %.1f%%)", item.TargetPct, item.BuyProbability*100, item.BuyBaseline*100),
		fmt.Sprintf("Падение ≤ -%.1f%%: %.1f%% (база %.1f%%)", item.TargetPct, item.SellProbability*100, item.SellBaseline*100),
	}
	f := item.Features
	if f.VolumeRatio >= 1.5 {
		reasons = append(reasons, fmt.Sprintf("Объём %.2fx от среднего", f.VolumeRatio))
	}
	if f.RSI14 >= 70 {
		reasons = append(reasons, fmt.Sprintf("RSI %.1f: зона повышенного импульса/перегрева", f.RSI14))
	} else if f.RSI14 <= 30 {
		reasons = append(reasons, fmt.Sprintf("RSI %.1f: зона сильной перепроданности", f.RSI14))
	}
	if f.Breakout20 > 0 {
		reasons = append(reasons, "Закрытие выше максимума предыдущих 20 свечей")
	} else if f.Breakout20 < 0 {
		reasons = append(reasons, "Закрытие ниже минимума предыдущих 20 свечей")
	}
	if math.Abs(f.EMAGap) >= 1 {
		direction := "восходящий"
		if f.EMAGap < 0 {
			direction = "нисходящий"
		}
		reasons = append(reasons, fmt.Sprintf("EMA20/EMA50: %s тренд (разрыв %.2f%%)", direction, f.EMAGap))
	}
	return reasons
}

func aggregateReasons(signal domain.Signal) []string {
	reasons := make([]string, 0, len(signal.Intervals)+2)
	for _, interval := range signal.Intervals {
		reasons = append(reasons, fmt.Sprintf("%s: %s, рост %.1f%% / падение %.1f%%, ожидание %+.2f%%", interval.Interval, interval.Action, interval.BuyProbability*100, interval.SellProbability*100, interval.ExpectedReturn))
	}
	reasons = append(reasons, fmt.Sprintf("Итоговый multi-timeframe score: %+.3f", signal.Score))
	return reasons
}

func futureExtremes(candles []domain.Candle, index, lookahead int) (float64, float64) {
	entry := candles[index].Close
	if entry == 0 || lookahead <= 0 {
		return 0, 0
	}
	maxReturn := -math.MaxFloat64
	minReturn := math.MaxFloat64
	end := index + lookahead
	if end >= len(candles) {
		end = len(candles) - 1
	}
	for i := index + 1; i <= end; i++ {
		highReturn := pctChange(entry, candles[i].High)
		lowReturn := pctChange(entry, candles[i].Low)
		if highReturn > maxReturn {
			maxReturn = highReturn
		}
		if lowReturn < minReturn {
			minReturn = lowReturn
		}
	}
	if maxReturn == -math.MaxFloat64 {
		maxReturn = 0
	}
	if minReturn == math.MaxFloat64 {
		minReturn = 0
	}
	return maxReturn, minReturn
}

func pctChange(from, to float64) float64 {
	if from == 0 {
		return 0
	}
	return (to - from) / from * 100
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
