package repository

import (
	"context"
	"encoding/json"

	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/domain"
)

func (s *Store) ListProjectJobs(ctx context.Context, projectID string) ([]domain.Job, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, COALESCE(project_id::text, ''), COALESCE(run_id::text, ''), job_type, status,
			stage, progress, message, payload::text, attempts, max_attempts, claimed_by, error_message
		FROM jobs
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT 80
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]domain.Job, 0)
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *Store) UpdateJobProgress(ctx context.Context, jobID, stage string, progress int, message, metadata string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var projectID, runID string
	if err := tx.QueryRow(ctx, `
		UPDATE jobs
		SET stage = $2, progress = $3, message = $4, heartbeat_at = now(), updated_at = now()
		WHERE id = $1
		RETURNING COALESCE(project_id::text, ''), COALESCE(run_id::text, '')
	`, jobID, stage, progress, message).Scan(&projectID, &runID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO job_events (job_id, project_id, run_id, event_type, stage, progress, message, metadata)
		VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, 'progress', $4, $5, $6, $7::jsonb)
	`, jobID, projectID, runID, stage, progress, message, jsonOrEmpty(metadata)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ListJobEvents(ctx context.Context, jobID string) ([]domain.JobEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, job_id, COALESCE(project_id::text, ''), COALESCE(run_id::text, ''), event_type,
			stage, progress, message, metadata::text, created_at
		FROM job_events
		WHERE job_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]domain.JobEvent, 0)
	for rows.Next() {
		var event domain.JobEvent
		if err := rows.Scan(&event.ID, &event.JobID, &event.ProjectID, &event.RunID, &event.EventType, &event.Stage, &event.Progress, &event.Message, &event.Metadata, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) EditSample(ctx context.Context, id, inputText, outputText, editedBy, reason, nextStatus string) (domain.Sample, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Sample{}, err
	}
	defer tx.Rollback(ctx)

	var version int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) + 1 FROM sample_versions WHERE sample_id = $1`, id).Scan(&version); err != nil {
		return domain.Sample{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO sample_versions (sample_id, version, input_text, output_text, edited_by, change_reason)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, version, inputText, outputText, editedBy, reason); err != nil {
		return domain.Sample{}, err
	}
	sample, err := scanSample(tx.QueryRow(ctx, `
		UPDATE samples
		SET input_text = $2, output_text = $3, status = $4, updated_at = now()
		WHERE id = $1
		RETURNING id, project_id, COALESCE(run_id::text, ''), COALESCE(source_id::text, ''),
			status, input_text, output_text, quality_score, token_count, created_at
	`, id, inputText, outputText, nextStatus))
	if err != nil {
		return domain.Sample{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO review_records (sample_id, action, reviewer, note) VALUES ($1, 'edited', $2, $3)`, id, editedBy, reason); err != nil {
		return domain.Sample{}, err
	}
	if sample.RunID != "" {
		if err := refreshRunSampleCounts(ctx, tx, sample.RunID); err != nil {
			return domain.Sample{}, err
		}
	}
	return sample, tx.Commit(ctx)
}

func (s *Store) ListSampleVersions(ctx context.Context, sampleID string) ([]domain.SampleVersion, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, sample_id, version, input_text, output_text, edited_by, change_reason, created_at
		FROM sample_versions
		WHERE sample_id = $1
		ORDER BY version DESC
	`, sampleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make([]domain.SampleVersion, 0)
	for rows.Next() {
		var version domain.SampleVersion
		if err := rows.Scan(&version.ID, &version.SampleID, &version.Version, &version.InputText, &version.OutputText, &version.EditedBy, &version.ChangeReason, &version.CreatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (s *Store) ListDatasetVersionFiles(ctx context.Context, versionID string) ([]domain.DatasetVersionFile, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, dataset_version_id, file_name, file_path, mime_type, byte_size, sha256, created_at
		FROM dataset_version_files
		WHERE dataset_version_id = $1
		ORDER BY created_at
	`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]domain.DatasetVersionFile, 0)
	for rows.Next() {
		file, err := scanDatasetVersionFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

func (s *Store) GetDatasetVersionFile(ctx context.Context, fileID string) (domain.DatasetVersionFile, error) {
	return scanDatasetVersionFile(s.pool.QueryRow(ctx, `
		SELECT id, dataset_version_id, file_name, file_path, mime_type, byte_size, sha256, created_at
		FROM dataset_version_files
		WHERE id = $1
	`, fileID))
}

func jsonOrEmpty(value string) string {
	var decoded any
	if value == "" || json.Unmarshal([]byte(value), &decoded) != nil {
		return "{}"
	}
	return value
}

func scanDatasetVersionFile(row scanner) (domain.DatasetVersionFile, error) {
	var file domain.DatasetVersionFile
	err := row.Scan(&file.ID, &file.DatasetVersionID, &file.FileName, &file.FilePath, &file.MimeType, &file.ByteSize, &file.SHA256, &file.CreatedAt)
	return file, err
}
