package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func pgSaveEstimate(ctx context.Context, ownerID string, estimate Estimate) error {
	pool, err := getDB(ctx)
	if err != nil {
		return err
	}
	if pool == nil {
		return fmt.Errorf("postgresql is not configured")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO estimates (
			id, owner_id, file_name, file_path, report_path, status, score,
			file_size, items_count, total_amount, uploaded_at, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now(),now())
	`, estimate.ID, ownerID, estimate.FileName, estimate.FilePath, estimate.ReportPath,
		estimate.Status, estimate.Score, estimate.FileSize, estimate.ItemsCount,
		estimate.TotalAmount, estimate.UploadedAt)
	if err != nil {
		return err
	}

	for _, item := range estimate.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO estimate_items (
				id, estimate_id, row_number, name, unit, quantity, unit_price, total
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		`, newDatabaseID("itm"), estimate.ID, item.Row, item.Name, item.Unit,
			item.Quantity, item.UnitPrice, item.Total)
		if err != nil {
			return err
		}
	}

	for _, finding := range estimate.Findings {
		_, err = tx.Exec(ctx, `
			INSERT INTO findings (id, estimate_id, title, severity, detail)
			VALUES ($1,$2,$3,$4,$5)
		`, newDatabaseID("fnd"), estimate.ID, finding.Title, finding.Severity, finding.Detail)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, action, resource_type, resource_id)
		VALUES ($1,$2,'estimate.uploaded','estimate',$3)
	`, newDatabaseID("aud"), ownerID, estimate.ID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func pgLoadEstimates(ctx context.Context, ownerID string) ([]Estimate, error) {
	pool, err := getDB(ctx)
	if err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, fmt.Errorf("postgresql is not configured")
	}

	rows, err := pool.Query(ctx, `
		SELECT id, owner_id, file_name, status, score, file_size, uploaded_at,
		       items_count, total_amount::float8, file_path, COALESCE(report_path, '')
		FROM estimates
		WHERE owner_id = $1
		ORDER BY uploaded_at DESC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	estimates := make([]Estimate, 0)
	for rows.Next() {
		estimate, err := scanEstimate(rows)
		if err != nil {
			return nil, err
		}
		if err := loadEstimateChildren(ctx, pool, &estimate); err != nil {
			return nil, err
		}
		estimates = append(estimates, estimate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return estimates, nil
}

func pgFindEstimate(ctx context.Context, ownerID, id string) (Estimate, bool, error) {
	pool, err := getDB(ctx)
	if err != nil {
		return Estimate{}, false, err
	}
	if pool == nil {
		return Estimate{}, false, fmt.Errorf("postgresql is not configured")
	}

	row := pool.QueryRow(ctx, `
		SELECT id, owner_id, file_name, status, score, file_size, uploaded_at,
		       items_count, total_amount::float8, file_path, COALESCE(report_path, '')
		FROM estimates
		WHERE id = $1 AND owner_id = $2
	`, id, ownerID)
	estimate, err := scanEstimate(row)
	if err == pgx.ErrNoRows {
		return Estimate{}, false, nil
	}
	if err != nil {
		return Estimate{}, false, err
	}
	if err := loadEstimateChildren(ctx, pool, &estimate); err != nil {
		return Estimate{}, false, err
	}
	return estimate, true, nil
}

func scanEstimate(scanner rowScanner) (Estimate, error) {
	var estimate Estimate
	var ownerID string
	err := scanner.Scan(
		&estimate.ID,
		&ownerID,
		&estimate.FileName,
		&estimate.Status,
		&estimate.Score,
		&estimate.FileSize,
		&estimate.UploadedAt,
		&estimate.ItemsCount,
		&estimate.TotalAmount,
		&estimate.FilePath,
		&estimate.ReportPath,
	)
	return estimate, err
}

func loadEstimateChildren(ctx context.Context, pool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, estimate *Estimate) error {
	itemRows, err := pool.Query(ctx, `
		SELECT row_number, COALESCE(name,''), COALESCE(unit,''),
		       quantity::float8, unit_price::float8, total::float8
		FROM estimate_items
		WHERE estimate_id = $1
		ORDER BY row_number
	`, estimate.ID)
	if err != nil {
		return err
	}
	defer itemRows.Close()

	estimate.Items = make([]EstimateItem, 0)
	for itemRows.Next() {
		var item EstimateItem
		if err := itemRows.Scan(&item.Row, &item.Name, &item.Unit, &item.Quantity, &item.UnitPrice, &item.Total); err != nil {
			return err
		}
		estimate.Items = append(estimate.Items, item)
	}
	if err := itemRows.Err(); err != nil {
		return err
	}

	findingRows, err := pool.Query(ctx, `
		SELECT title, severity, detail
		FROM findings
		WHERE estimate_id = $1
		ORDER BY created_at, id
	`, estimate.ID)
	if err != nil {
		return err
	}
	defer findingRows.Close()

	estimate.Findings = make([]Finding, 0)
	for findingRows.Next() {
		var finding Finding
		if err := findingRows.Scan(&finding.Title, &finding.Severity, &finding.Detail); err != nil {
			return err
		}
		estimate.Findings = append(estimate.Findings, finding)
	}
	return findingRows.Err()
}

func pgSaveCompareResult(ctx context.Context, ownerID string, result CompareResponse) error {
	pool, err := getDB(ctx)
	if err != nil {
		return err
	}
	if pool == nil {
		return fmt.Errorf("postgresql is not configured")
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO compare_results (
			id, owner_id, base_file_name, new_file_name, base_total, new_total,
			delta_total, added_count, removed_count, changed_count, payload, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, result.ID, ownerID, result.BaseFile, result.NewFile, result.BaseTotal,
		result.NewTotal, result.DeltaTotal, len(result.Added), len(result.Removed),
		len(result.Changed), payload, result.CreatedAt)
	return err
}

func pgDeleteEstimate(ctx context.Context, ownerID, id string) (string, string, bool, error) {
	pool, err := getDB(ctx)
	if err != nil {
		return "", "", false, err
	}
	if pool == nil {
		return "", "", false, fmt.Errorf("postgresql is not configured")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", "", false, err
	}
	defer tx.Rollback(ctx)

	var filePath, reportPath string
	err = tx.QueryRow(ctx, `
		DELETE FROM estimates
		WHERE id=$1 AND owner_id=$2
		RETURNING COALESCE(file_path,''), COALESCE(report_path,'')
	`, id, ownerID).Scan(&filePath, &reportPath)
	if err == pgx.ErrNoRows {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, action, resource_type, resource_id)
		VALUES ($1,$2,'estimate.deleted','estimate',$3)
	`, newDatabaseID("aud"), ownerID, id)
	if err != nil {
		return "", "", false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", "", false, err
	}
	return filePath, reportPath, true, nil
}

func newDatabaseID(prefix string) string {
	return prefix + "_" + uuid.NewString()
}
