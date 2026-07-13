package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/analyzer"
	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/binance"
	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/config"
	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/domain"
	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/store"
	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/telegram"
)

type Service struct {
	cfg      config.Config
	store    *store.Store
	binance  *binance.Client
	engine   analyzer.Engine
	telegram *telegram.Client
	logger   *slog.Logger
	scanMu   sync.Mutex
}

func New(cfg config.Config, database *store.Store, market *binance.Client, bot *telegram.Client, logger *slog.Logger) *Service {
	return &Service{
		cfg: cfg, store: database, binance: market, telegram: bot, logger: logger,
		engine: analyzer.Engine{
			NearestNeighbors: cfg.NearestNeighbors, MinimumSamples: cfg.MinimumSamples,
			MinimumProbability: cfg.MinimumProbability, MinimumEdge: cfg.MinimumEdge,
			SignalThreshold: cfg.SignalThreshold, Lookahead: cfg.LookaheadByInterval,
			Targets: cfg.TargetByInterval, Weights: cfg.WeightByInterval,
		},
	}
}

func (s *Service) Start(ctx context.Context) {
	if s.cfg.RunOnStartup {
		go func() {
			if err := s.Scan(ctx); err != nil {
				s.logger.Error("startup scan failed", "error", err)
			}
		}()
	}
	go s.scheduler(ctx)
	if s.cfg.TelegramPolling && s.telegram.Enabled() {
		go s.telegramCommands(ctx)
	}
}

func (s *Service) Scan(ctx context.Context) error {
	if !s.scanMu.TryLock() {
		return fmt.Errorf("scan is already running")
	}
	defer s.scanMu.Unlock()

	startedAt := time.Now().UTC()
	s.logger.Info("market scan started")
	symbols, err := s.binance.Symbols(ctx, s.cfg.QuoteAsset, s.cfg.MinQuoteVolume, s.cfg.MaxSymbols, s.cfg.ExcludedBaseAssets)
	if err != nil {
		return err
	}
	if len(symbols) == 0 {
		return fmt.Errorf("Binance returned no symbols after filters")
	}
	if err := s.store.UpsertSymbols(ctx, symbols); err != nil {
		return fmt.Errorf("save symbols: %w", err)
	}
	runID, err := s.store.StartScan(ctx, len(symbols))
	if err != nil {
		return fmt.Errorf("create scan run: %w", err)
	}
	status := "failed"
	analyzedCount := 0
	signalCount := 0
	errorMessage := "scan interrupted"
	defer func() {
		finishCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if finishErr := s.store.FinishScan(finishCtx, runID, status, analyzedCount, signalCount, errorMessage); finishErr != nil {
			s.logger.Error("finish scan record failed", "scan_run_id", runID, "error", finishErr)
		}
	}()

	type result struct {
		signal domain.Signal
		valid  bool
	}
	jobs := make(chan domain.Symbol)
	results := make(chan result, len(symbols))
	workerCount := s.cfg.RequestConcurrency
	if workerCount > len(symbols) {
		workerCount = len(symbols)
	}
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for symbol := range jobs {
				signal, ok := s.analyzeSymbol(ctx, runID, symbol)
				if ok {
					results <- result{signal: signal, valid: true}
				}
			}
		}(i)
	}
	go func() {
		defer close(jobs)
		for _, symbol := range symbols {
			select {
			case <-ctx.Done():
				return
			case jobs <- symbol:
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	allSignals := make([]domain.Signal, 0, len(symbols))
	for item := range results {
		if item.valid {
			analyzedCount++
			allSignals = append(allSignals, item.signal)
		}
	}
	if ctx.Err() != nil {
		errorMessage = ctx.Err().Error()
		return ctx.Err()
	}
	if analyzedCount == 0 {
		errorMessage = "no symbols were successfully analyzed"
		return fmt.Errorf("%s", errorMessage)
	}

	s.enrichFutures(ctx, allSignals)
	if err := s.store.SaveSignals(ctx, runID, allSignals); err != nil {
		errorMessage = "save signals: " + err.Error()
		return fmt.Errorf("%s", errorMessage)
	}
	for _, signal := range allSignals {
		if signal.Action != "HOLD" {
			signalCount++
		}
	}
	status = "completed"
	errorMessage = ""
	finishedAt := time.Now().UTC()
	run := domain.ScanRun{
		ID: runID, StartedAt: startedAt, FinishedAt: finishedAt, Status: status,
		SymbolsSelected: len(symbols), SymbolsAnalyzed: analyzedCount, SignalsCreated: signalCount,
	}

	if s.telegram.Enabled() && (signalCount > 0 || s.cfg.TelegramSendHolds) {
		if err := s.sendReport(ctx, run, allSignals, "hourly_report"); err != nil {
			s.logger.Error("Telegram report failed", "scan_run_id", runID, "error", err)
		}
	}
	s.logger.Info("market scan completed", "scan_run_id", runID, "selected", len(symbols), "analyzed", analyzedCount, "signals", signalCount, "duration", time.Since(startedAt))
	return nil
}

func (s *Service) analyzeSymbol(ctx context.Context, runID int64, symbol domain.Symbol) (domain.Signal, bool) {
	analyses := make([]domain.IntervalAnalysis, 0, len(s.cfg.Intervals))
	for _, interval := range s.cfg.Intervals {
		candles, err := s.binance.Klines(ctx, symbol.Symbol, interval, s.cfg.KlineLimit)
		if err != nil {
			s.logger.Warn("kline download failed", "symbol", symbol.Symbol, "interval", interval, "error", err)
			continue
		}
		if err := s.store.UpsertCandles(ctx, candles); err != nil {
			s.logger.Warn("candle save failed", "symbol", symbol.Symbol, "interval", interval, "error", err)
			continue
		}
		analysisResult, err := s.engine.AnalyzeInterval(symbol.Symbol, interval, candles)
		if err != nil {
			s.logger.Debug("interval analysis skipped", "symbol", symbol.Symbol, "interval", interval, "error", err)
			continue
		}
		if err := s.store.SaveIntervalAnalysis(ctx, runID, analysisResult); err != nil {
			s.logger.Warn("interval analysis save failed", "symbol", symbol.Symbol, "interval", interval, "error", err)
			continue
		}
		analyses = append(analyses, analysisResult)
	}
	minimumIntervals := 2
	if len(s.cfg.Intervals) == 1 {
		minimumIntervals = 1
	}
	if len(analyses) < minimumIntervals {
		return domain.Signal{}, false
	}
	return s.engine.Aggregate(symbol.Symbol, analyses), true
}

func (s *Service) enrichFutures(ctx context.Context, signals []domain.Signal) {
	if !s.cfg.EnableFutures || s.cfg.FuturesEnrichLimit <= 0 {
		return
	}
	indices := make([]int, 0, len(signals))
	for i := range signals {
		if signals[i].Action != "HOLD" || math.Abs(signals[i].Score) >= s.cfg.SignalThreshold*0.75 {
			indices = append(indices, i)
		}
	}
	sort.Slice(indices, func(i, j int) bool {
		return math.Abs(signals[indices[i]].Score) > math.Abs(signals[indices[j]].Score)
	})
	if len(indices) > s.cfg.FuturesEnrichLimit {
		indices = indices[:s.cfg.FuturesEnrichLimit]
	}

	sem := make(chan struct{}, min(4, s.cfg.RequestConcurrency))
	var wg sync.WaitGroup
	for _, index := range indices {
		index := index
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			metrics, err := s.binance.FuturesMetrics(ctx, signals[index].Symbol)
			if err != nil {
				return
			}
			analyzer.ApplyFuturesMetrics(&signals[index], metrics)
		}()
	}
	wg.Wait()
}

func (s *Service) scheduler(ctx context.Context) {
	for {
		now := time.Now().UTC()
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), s.cfg.ScanMinute, 0, 0, time.UTC)
		if !next.After(now) {
			next = next.Add(time.Hour)
		}
		timer := time.NewTimer(time.Until(next))
		s.logger.Info("next scan scheduled", "at", next)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if err := s.Scan(ctx); err != nil {
				s.logger.Error("scheduled scan failed", "error", err)
			}
		}
	}
}

func (s *Service) telegramCommands(ctx context.Context) {
	var offset atomic.Int64
	for ctx.Err() == nil {
		updates, err := s.telegram.GetUpdates(ctx, offset.Load())
		if err != nil {
			s.logger.Warn("Telegram long polling failed", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}
		for _, update := range updates {
			offset.Store(update.UpdateID + 1)
			if update.Message == nil || fmt.Sprint(update.Message.Chat.ID) != s.telegram.ChatID() {
				continue
			}
			text := strings.TrimSpace(update.Message.Text)
			response := ""
			switch {
			case text == "/report" || strings.HasPrefix(text, "/report@"):
				run, _ := s.store.LatestScan(ctx)
				signals, _ := s.store.LatestSignals(ctx, 200)
				location, _ := time.LoadLocation("Asia/Bishkek")
				response = telegram.FormatReport(s.cfg.AppName, run, signals, s.cfg.ReportTopN, location)
			case text == "/status" || strings.HasPrefix(text, "/status@"):
				run, _ := s.store.LatestScan(ctx)
				response = telegram.FormatStatus(run)
			case strings.HasPrefix(text, "/coin"):
				parts := strings.Fields(text)
				if len(parts) < 2 {
					response = "Использование: /coin BTCUSDT"
				} else {
					symbol := strings.ToUpper(strings.TrimSpace(parts[1]))
					signals, _ := s.store.LatestSignalsForSymbol(ctx, symbol, 20)
					response = telegram.FormatCoin(symbol, signals)
				}
			case text == "/scan" || strings.HasPrefix(text, "/scan@"):
				response = "Сканирование запущено. Результат придёт отдельным отчётом."
				go func() {
					if err := s.Scan(ctx); err != nil {
						_, _ = s.telegram.Send(context.Background(), "Сканирование не выполнено: "+err.Error())
					}
				}()
			default:
				response = telegram.HelpText() + "\n/scan — запустить сканирование сейчас"
			}
			if response != "" {
				_, _ = s.telegram.Send(ctx, response)
			}
		}
	}
}

func (s *Service) sendReport(ctx context.Context, run domain.ScanRun, signals []domain.Signal, kind string) error {
	location, err := time.LoadLocation("Asia/Bishkek")
	if err != nil {
		location = time.FixedZone("Asia/Bishkek", 6*60*60)
	}
	message := telegram.FormatReport(s.cfg.AppName, run, signals, s.cfg.ReportTopN, location)
	code, sendErr := s.telegram.Send(ctx, message)
	recordCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	errorText := ""
	if sendErr != nil {
		errorText = sendErr.Error()
	}
	_ = s.store.RecordTelegramDelivery(recordCtx, run.ID, s.telegram.ChatID(), kind, sendErr == nil, code, errorText)
	return sendErr
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
