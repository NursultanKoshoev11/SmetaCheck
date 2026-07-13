package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/domain"
)

type Client struct {
	spotBase    string
	futuresBase string
	httpClient  *http.Client
	delay       time.Duration
}

func New(spotBase, futuresBase string, timeout, delay time.Duration) *Client {
	return &Client{
		spotBase:    strings.TrimRight(spotBase, "/"),
		futuresBase: strings.TrimRight(futuresBase, "/"),
		httpClient:  &http.Client{Timeout: timeout},
		delay:       delay,
	}
}

type exchangeInfoResponse struct {
	Symbols []struct {
		Symbol               string `json:"symbol"`
		Status               string `json:"status"`
		BaseAsset            string `json:"baseAsset"`
		QuoteAsset           string `json:"quoteAsset"`
		IsSpotTradingAllowed bool   `json:"isSpotTradingAllowed"`
	} `json:"symbols"`
}

type ticker24h struct {
	Symbol      string `json:"symbol"`
	QuoteVolume string `json:"quoteVolume"`
}

func (c *Client) Symbols(ctx context.Context, quoteAsset string, minQuoteVolume float64, maxSymbols int, excluded map[string]struct{}) ([]domain.Symbol, error) {
	var info exchangeInfoResponse
	if err := c.getJSON(ctx, c.spotBase+"/api/v3/exchangeInfo", &info); err != nil {
		return nil, fmt.Errorf("exchange info: %w", err)
	}
	var tickers []ticker24h
	if err := c.getJSON(ctx, c.spotBase+"/api/v3/ticker/24hr", &tickers); err != nil {
		return nil, fmt.Errorf("24h tickers: %w", err)
	}
	volumeBySymbol := make(map[string]float64, len(tickers))
	for _, ticker := range tickers {
		volumeBySymbol[ticker.Symbol] = parseFloat(ticker.QuoteVolume)
	}

	quoteAsset = strings.ToUpper(quoteAsset)
	result := make([]domain.Symbol, 0, len(info.Symbols))
	for _, item := range info.Symbols {
		if item.Status != "TRADING" || !item.IsSpotTradingAllowed || item.QuoteAsset != quoteAsset {
			continue
		}
		if isLeveragedToken(item.BaseAsset) {
			continue
		}
		if _, skip := excluded[strings.ToUpper(item.BaseAsset)]; skip {
			continue
		}
		quoteVolume := volumeBySymbol[item.Symbol]
		if quoteVolume < minQuoteVolume {
			continue
		}
		result = append(result, domain.Symbol{
			Symbol:      item.Symbol,
			BaseAsset:   item.BaseAsset,
			QuoteAsset:  item.QuoteAsset,
			QuoteVolume: quoteVolume,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].QuoteVolume > result[j].QuoteVolume })
	if maxSymbols > 0 && len(result) > maxSymbols {
		result = result[:maxSymbols]
	}
	return result, nil
}

func (c *Client) Klines(ctx context.Context, symbol, interval string, limit int) ([]domain.Candle, error) {
	query := url.Values{}
	query.Set("symbol", symbol)
	query.Set("interval", interval)
	query.Set("limit", strconv.Itoa(limit))
	endpoint := c.spotBase + "/api/v3/klines?" + query.Encode()

	var rows [][]json.RawMessage
	if err := c.getJSON(ctx, endpoint, &rows); err != nil {
		return nil, fmt.Errorf("klines %s %s: %w", symbol, interval, err)
	}

	now := time.Now().UTC()
	candles := make([]domain.Candle, 0, len(rows))
	for _, row := range rows {
		if len(row) < 11 {
			continue
		}
		openMS := rawInt64(row[0])
		closeMS := rawInt64(row[6])
		closeTime := time.UnixMilli(closeMS).UTC()
		if !closeTime.Before(now) {
			continue // Never analyze an unfinished candle.
		}
		candles = append(candles, domain.Candle{
			Symbol:         symbol,
			Interval:       interval,
			OpenTime:       time.UnixMilli(openMS).UTC(),
			CloseTime:      closeTime,
			Open:           rawFloat(row[1]),
			High:           rawFloat(row[2]),
			Low:            rawFloat(row[3]),
			Close:          rawFloat(row[4]),
			Volume:         rawFloat(row[5]),
			QuoteVolume:    rawFloat(row[7]),
			Trades:         rawInt64(row[8]),
			TakerBuyVolume: rawFloat(row[9]),
		})
	}
	return candles, nil
}

func (c *Client) FuturesMetrics(ctx context.Context, symbol string) (domain.FuturesMetrics, error) {
	metrics := domain.FuturesMetrics{FetchedAt: time.Now().UTC()}

	var oi struct {
		OpenInterest string `json:"openInterest"`
		Time         int64  `json:"time"`
	}
	query := url.Values{"symbol": []string{symbol}}
	if err := c.getJSON(ctx, c.futuresBase+"/fapi/v1/openInterest?"+query.Encode(), &oi); err != nil {
		return metrics, err
	}

	var funding []struct {
		FundingRate string `json:"fundingRate"`
		FundingTime int64  `json:"fundingTime"`
	}
	query.Set("limit", "1")
	if err := c.getJSON(ctx, c.futuresBase+"/fapi/v1/fundingRate?"+query.Encode(), &funding); err != nil {
		return metrics, err
	}

	metrics.Available = true
	metrics.OpenInterest = parseFloat(oi.OpenInterest)
	if len(funding) > 0 {
		metrics.FundingRate = parseFloat(funding[len(funding)-1].FundingRate)
	}

	// The historical endpoint is limited to recent data, so use two recent points
	// only as a short-term change feature rather than pretending it is long history.
	var history []struct {
		SumOpenInterest string `json:"sumOpenInterest"`
		Timestamp       int64  `json:"timestamp"`
	}
	historyQuery := url.Values{}
	historyQuery.Set("symbol", symbol)
	historyQuery.Set("period", "1h")
	historyQuery.Set("limit", "2")
	if err := c.getJSON(ctx, c.futuresBase+"/futures/data/openInterestHist?"+historyQuery.Encode(), &history); err == nil && len(history) >= 2 {
		oldValue := parseFloat(history[len(history)-2].SumOpenInterest)
		newValue := parseFloat(history[len(history)-1].SumOpenInterest)
		if oldValue != 0 {
			metrics.OpenInterestPct = (newValue - oldValue) / oldValue * 100
		}
	}
	return metrics, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, dst any) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
		if c.delay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.delay):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "crypto-pattern-analyzer/1.0")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
			if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
				continue
			}
			return lastErr
		}
		if err := json.Unmarshal(body, dst); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New("request failed")
	}
	return lastErr
}

func isLeveragedToken(base string) bool {
	upper := strings.ToUpper(base)
	for _, suffix := range []string{"UP", "DOWN", "BULL", "BEAR"} {
		if strings.HasSuffix(upper, suffix) {
			return true
		}
	}
	return false
}

func rawFloat(raw json.RawMessage) float64 {
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return parseFloat(value)
	}
	var number float64
	_ = json.Unmarshal(raw, &number)
	return number
}

func rawInt64(raw json.RawMessage) int64 {
	var value int64
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		value, _ = strconv.ParseInt(text, 10, 64)
	}
	return value
}

func parseFloat(value string) float64 {
	number, _ := strconv.ParseFloat(value, 64)
	return number
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
