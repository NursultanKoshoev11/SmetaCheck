package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/domain"
	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	config.MaxConns = 10
	config.MinConns = 1
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 15 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	result := &Store{pool: pool}
	if err := result.Migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return result, nil
}

func (s *Store) Close() { s.pool.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }
func (s *Store) Migrate(ctx context.Context) error { if _, err := s.pool.Exec(ctx, migrations.InitialSQL); err != nil { return fmt.Errorf("apply database schema: %w", err) }; return nil }
func (s *Store) StartScan(ctx context.Context, selected int) (int64, error) { var id int64; err := s.pool.QueryRow(ctx, `INSERT INTO crypto_scan_runs(status, symbols_selected) VALUES ('running', $1) RETURNING id`, selected).Scan(&id); return id, err }
func (s *Store) FinishScan(ctx context.Context, id int64, status string, analyzed, signals int, errorMessage string) error { _, err := s.pool.Exec(ctx, `UPDATE crypto_scan_runs SET finished_at=now(), status=$2, symbols_analyzed=$3, signals_created=$4, error_message=$5 WHERE id=$1`, id, status, analyzed, signals, errorMessage); return err }

func (s *Store) UpsertSymbols(ctx context.Context, symbols []domain.Symbol) error {
	batch := &pgx.Batch{}
	for _, symbol := range symbols { batch.Queue(`INSERT INTO crypto_symbols(symbol,base_asset,quote_asset,quote_volume_24h,active,last_seen_at) VALUES($1,$2,$3,$4,true,now()) ON CONFLICT(symbol) DO UPDATE SET base_asset=EXCLUDED.base_asset,quote_asset=EXCLUDED.quote_asset,quote_volume_24h=EXCLUDED.quote_volume_24h,active=true,last_seen_at=now()`, symbol.Symbol,symbol.BaseAsset,symbol.QuoteAsset,symbol.QuoteVolume) }
	results := s.pool.SendBatch(ctx,batch); defer results.Close(); for range symbols { if _,err:=results.Exec(); err!=nil{return err} }; return nil
}

func (s *Store) UpsertCandles(ctx context.Context, candles []domain.Candle) error {
	if len(candles)==0{return nil}; batch:=&pgx.Batch{}
	for _,c:=range candles { batch.Queue(`INSERT INTO crypto_candles(symbol,interval,open_time,close_time,open,high,low,close,volume,quote_volume,trades,taker_buy_volume) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT(symbol,interval,open_time) DO UPDATE SET close_time=EXCLUDED.close_time,open=EXCLUDED.open,high=EXCLUDED.high,low=EXCLUDED.low,close=EXCLUDED.close,volume=EXCLUDED.volume,quote_volume=EXCLUDED.quote_volume,trades=EXCLUDED.trades,taker_buy_volume=EXCLUDED.taker_buy_volume`,c.Symbol,c.Interval,c.OpenTime,c.CloseTime,c.Open,c.High,c.Low,c.Close,c.Volume,c.QuoteVolume,c.Trades,c.TakerBuyVolume) }
	results:=s.pool.SendBatch(ctx,batch); defer results.Close(); for range candles {if _,err:=results.Exec();err!=nil{return err}}; return nil
}

func (s *Store) SaveIntervalAnalysis(ctx context.Context, scanRunID int64, item domain.IntervalAnalysis) error {
	features,err:=json.Marshal(item.Features);if err!=nil{return err};reasons,err:=json.Marshal(item.Reasons);if err!=nil{return err}
	_,err=s.pool.Exec(ctx,`INSERT INTO crypto_interval_analyses(scan_run_id,symbol,interval,candle_open_time,price,action,buy_probability,sell_probability,buy_baseline,sell_baseline,expected_return,sample_count,nearest_count,average_distance,score,confidence,lookahead_bars,target_pct,features,reasons) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20) ON CONFLICT(scan_run_id,symbol,interval) DO UPDATE SET price=EXCLUDED.price,action=EXCLUDED.action,buy_probability=EXCLUDED.buy_probability,sell_probability=EXCLUDED.sell_probability,buy_baseline=EXCLUDED.buy_baseline,sell_baseline=EXCLUDED.sell_baseline,expected_return=EXCLUDED.expected_return,sample_count=EXCLUDED.sample_count,nearest_count=EXCLUDED.nearest_count,average_distance=EXCLUDED.average_distance,score=EXCLUDED.score,confidence=EXCLUDED.confidence,features=EXCLUDED.features,reasons=EXCLUDED.reasons`,scanRunID,item.Symbol,item.Interval,item.CandleOpenTime,item.Price,item.Action,item.BuyProbability,item.SellProbability,item.BuyBaseline,item.SellBaseline,item.ExpectedReturn,item.SampleCount,item.NearestCount,item.Distance,item.Score,item.Confidence,item.LookaheadBars,item.TargetPct,features,reasons);return err
}

func (s *Store) SaveSignals(ctx context.Context, scanRunID int64, signals []domain.Signal) error {
	batch:=&pgx.Batch{};for _,signal:=range signals{reasons,err:=json.Marshal(signal.Reasons);if err!=nil{return err};batch.Queue(`INSERT INTO crypto_signals(scan_run_id,symbol,action,price,score,confidence,expected_return,sample_count,funding_rate,open_interest,open_interest_pct,reasons) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT(scan_run_id,symbol) DO UPDATE SET action=EXCLUDED.action,price=EXCLUDED.price,score=EXCLUDED.score,confidence=EXCLUDED.confidence,expected_return=EXCLUDED.expected_return,sample_count=EXCLUDED.sample_count,funding_rate=EXCLUDED.funding_rate,open_interest=EXCLUDED.open_interest,open_interest_pct=EXCLUDED.open_interest_pct,reasons=EXCLUDED.reasons`,scanRunID,signal.Symbol,signal.Action,signal.Price,signal.Score,signal.Confidence,signal.ExpectedReturn,signal.SampleCount,signal.FundingRate,signal.OpenInterest,signal.OpenInterestPct,reasons)};results:=s.pool.SendBatch(ctx,batch);defer results.Close();for range signals{if _,err:=results.Exec();err!=nil{return err}};return nil
}

func (s *Store) LatestSignals(ctx context.Context, limit int) ([]domain.Signal,error){if limit<=0||limit>200{limit=20};rows,err:=s.pool.Query(ctx,`SELECT id,scan_run_id,symbol,action,price,score,confidence,expected_return,sample_count,funding_rate,open_interest,open_interest_pct,reasons,created_at FROM crypto_signals WHERE scan_run_id=(SELECT id FROM crypto_scan_runs WHERE status='completed' ORDER BY id DESC LIMIT 1) ORDER BY ABS(score) DESC,confidence DESC LIMIT $1`,limit);if err!=nil{return nil,err};defer rows.Close();return scanSignals(rows)}
func (s *Store) LatestSignalsForSymbol(ctx context.Context,symbol string,limit int)([]domain.Signal,error){if limit<=0||limit>200{limit=20};rows,err:=s.pool.Query(ctx,`SELECT id,scan_run_id,symbol,action,price,score,confidence,expected_return,sample_count,funding_rate,open_interest,open_interest_pct,reasons,created_at FROM crypto_signals WHERE symbol=$1 ORDER BY created_at DESC LIMIT $2`,symbol,limit);if err!=nil{return nil,err};defer rows.Close();return scanSignals(rows)}
func (s *Store) LatestScan(ctx context.Context)(domain.ScanRun,error){var run domain.ScanRun;err:=s.pool.QueryRow(ctx,`SELECT id,started_at,COALESCE(finished_at,started_at),status,symbols_selected,symbols_analyzed,signals_created,error_message FROM crypto_scan_runs ORDER BY id DESC LIMIT 1`).Scan(&run.ID,&run.StartedAt,&run.FinishedAt,&run.Status,&run.SymbolsSelected,&run.SymbolsAnalyzed,&run.SignalsCreated,&run.ErrorMessage);if errors.Is(err,pgx.ErrNoRows){return domain.ScanRun{},nil};return run,err}
func (s *Store) RecordTelegramDelivery(ctx context.Context,scanRunID int64,chatID,kind string,success bool,responseCode int,errorMessage string)error{var id any=scanRunID;if scanRunID==0{id=nil};_,err:=s.pool.Exec(ctx,`INSERT INTO crypto_telegram_deliveries(scan_run_id,chat_id,message_kind,success,response_code,error_message) VALUES($1,$2,$3,$4,$5,$6)`,id,chatID,kind,success,responseCode,errorMessage);return err}

type rowsScanner interface{Next()bool;Scan(dest ...any)error;Err()error}
func scanSignals(rows rowsScanner)([]domain.Signal,error){result:=make([]domain.Signal,0);for rows.Next(){var signal domain.Signal;var reasons []byte;if err:=rows.Scan(&signal.ID,&signal.ScanRunID,&signal.Symbol,&signal.Action,&signal.Price,&signal.Score,&signal.Confidence,&signal.ExpectedReturn,&signal.SampleCount,&signal.FundingRate,&signal.OpenInterest,&signal.OpenInterestPct,&reasons,&signal.CreatedAt);err!=nil{return nil,err};_ = json.Unmarshal(reasons,&signal.Reasons);result=append(result,signal)};return result,rows.Err()}
