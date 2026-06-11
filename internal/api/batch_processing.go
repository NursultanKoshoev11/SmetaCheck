package api

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// ProcessNextAnalysisBatch atomically claims and processes one pending batch.
// It returns true when a batch was claimed, including when processing failed.
func ProcessNextAnalysisBatch(ctx context.Context) (bool, error) {
	batch, ownerID, found, err := pgClaimNextAnalysisBatch(ctx)
	if err != nil || !found {
		return found, err
	}
	if err := processClaimedAnalysisBatch(ctx, ownerID, batch); err != nil {
		_ = pgFailAnalysisBatch(ctx, ownerID, batch.ID, err.Error())
		return true, err
	}
	return true, nil
}

func processClaimedAnalysisBatch(ctx context.Context, ownerID string, batch AnalysisBatch) error {
	files, err := pgLoadAnalysisBatchFiles(ctx, batch.ID)
	if err != nil {
		return fmt.Errorf("load batch files: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("analysis batch contains no files")
	}

	estimates := make([]Estimate, 0, len(files))
	completed := 0
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		_ = pgUpdateBatchFile(ctx, file.ID, "processing", file.EstimateID, "")
		estimate, err := ensureBatchEstimate(ctx, ownerID, file)
		if err != nil {
			_ = pgUpdateBatchFile(ctx, file.ID, "failed", file.EstimateID, err.Error())
			return fmt.Errorf("process %s: %w", file.FileName, err)
		}
		estimates = append(estimates, estimate)
		completed++
		if err := pgUpdateBatchFile(ctx, file.ID, "completed", estimate.ID, ""); err != nil {
			return err
		}
		if err := pgTouchAnalysisBatch(ctx, batch.ID, completed); err != nil {
			return err
		}
	}

	provider, err := configuredAIProvider(batch.Provider, batch.Model)
	if err != nil {
		return fmt.Errorf("configure AI provider: %w", err)
	}
	aiResult, err := provider.AnalyzeBatch(ctx, buildAIProviderInputs(estimates))
	if err != nil {
		return fmt.Errorf("AI batch analysis failed: %w", err)
	}
	aiResult.Provider = provider.Name()
	aiResult.Model = provider.Model()
	if aiResult.GeneratedAt.IsZero() {
		aiResult.GeneratedAt = time.Now().UTC()
	}

	report := reconcileBatch(batch.ID, estimates, aiResult)
	if err := pgCompleteAnalysisBatch(ctx, ownerID, batch.ID, report); err != nil {
		return fmt.Errorf("save unified report: %w", err)
	}
	return nil
}

func ensureBatchEstimate(ctx context.Context, ownerID string, file AnalysisBatchFile) (Estimate, error) {
	estimateID := file.EstimateID
	if estimateID == "" {
		estimateID = stableBatchEstimateID(file.ID)
	}
	if existing, found, err := pgFindEstimate(ctx, ownerID, estimateID); err != nil {
		return Estimate{}, err
	} else if found {
		return existing, nil
	}

	items, findings := analyzeEstimateFile(file.FilePath, file.FileName, file.FileSize)
	reportDir := estimateReportDir()
	estimate := Estimate{
		ID:          estimateID,
		FileName:    file.FileName,
		Status:      estimateStatus(findings),
		Score:       scoreFromFindings(findings),
		FileSize:    file.FileSize,
		UploadedAt:  time.Now().UTC(),
		ItemsCount:  len(items),
		TotalAmount: sumItemsTotal(items),
		Findings:    findings,
		Items:       items,
		FilePath:    file.FilePath,
		ReportPath:  filepath.Join(reportDir, estimateID+"_report.txt"),
	}
	if err := writeEstimateReport(estimate); err != nil {
		return Estimate{}, fmt.Errorf("create backend report: %w", err)
	}
	if err := pgSaveEstimate(ctx, ownerID, estimate); err != nil {
		// A previous worker may have committed the stable estimate before losing its lock.
		if existing, found, findErr := pgFindEstimate(ctx, ownerID, estimateID); findErr == nil && found {
			return existing, nil
		}
		return Estimate{}, fmt.Errorf("save backend analysis: %w", err)
	}
	return estimate, nil
}

func stableBatchEstimateID(fileID string) string {
	clean := strings.NewReplacer("-", "", "_", "").Replace(fileID)
	if len(clean) > 48 {
		clean = clean[len(clean)-48:]
	}
	return "est_batch_" + clean
}
