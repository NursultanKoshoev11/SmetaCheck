package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func pgLoadAIReport(ctx context.Context, ownerID, estimateID, inputHash string) (AISummaryResponse, bool, error) {
	pool, err := getDB(ctx)
	if err != nil {
		return AISummaryResponse{}, false, err
	}
	if pool == nil {
		return AISummaryResponse{}, false, fmt.Errorf("postgresql is not configured")
	}

	var payload []byte
	err = pool.QueryRow(ctx, `
		SELECT payload
		FROM ai_reports
		WHERE owner_id = $1
		  AND estimate_id = $2
		  AND input_hash = $3
		  AND provider = 'openai'
		  AND model = $4
		  AND prompt_version = $5
		ORDER BY created_at DESC
		LIMIT 1
	`, ownerID, estimateID, inputHash, openAIModel(), aiPromptVersion).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return AISummaryResponse{}, false, nil
	}
	if err != nil {
		return AISummaryResponse{}, false, err
	}

	var report AISummaryResponse
	if err := json.Unmarshal(payload, &report); err != nil {
		return AISummaryResponse{}, false, fmt.Errorf("decode cached AI report: %w", err)
	}
	return report, true, nil
}

func pgSaveAIReport(ctx context.Context, ownerID, inputHash string, report AISummaryResponse) error {
	pool, err := getDB(ctx)
	if err != nil {
		return err
	}
	if pool == nil {
		return fmt.Errorf("postgresql is not configured")
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("encode AI report: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO ai_reports (
			id, owner_id, estimate_id, input_hash, provider, model,
			prompt_version, payload, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (owner_id, estimate_id, input_hash, provider, model, prompt_version)
		DO UPDATE SET payload = EXCLUDED.payload, created_at = EXCLUDED.created_at
	`, newDatabaseID("air"), ownerID, report.EstimateID, inputHash, report.Provider,
		report.Model, report.PromptVersion, payload, report.GeneratedAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, action, resource_type, resource_id)
		VALUES ($1,$2,'estimate.ai_report.generated','estimate',$3)
	`, newDatabaseID("aud"), ownerID, report.EstimateID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
