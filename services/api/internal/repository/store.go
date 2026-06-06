package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) ApplyMigrations(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, string(body)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateProject(ctx context.Context, name, description, domainName string) (domain.Project, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO projects (name, description, domain)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, domain, created_at, updated_at
	`, name, description, domainName)
	return scanProject(row)
}

func (s *Store) ListProjects(ctx context.Context) ([]domain.Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, domain, created_at, updated_at
		FROM projects
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

func (s *Store) GetProject(ctx context.Context, id string) (domain.Project, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, name, description, domain, created_at, updated_at
		FROM projects
		WHERE id = $1
	`, id)
	return scanProject(row)
}

func (s *Store) CreateSource(ctx context.Context, source domain.Source) (domain.Source, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO sources (
			project_id, filename, source_type, status, artifact_path, content_hash, byte_size
		)
		VALUES ($1, $2, $3, 'ready', $4, $5, $6)
		RETURNING id, project_id, filename, source_type, status, artifact_path, content_hash, byte_size, created_at
	`, source.ProjectID, source.Filename, source.SourceType, source.ArtifactPath, source.ContentHash, source.ByteSize)
	return scanSource(row)
}

func (s *Store) ListSources(ctx context.Context, projectID string) ([]domain.Source, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, project_id, filename, source_type, status, artifact_path, content_hash, byte_size, created_at
		FROM sources
		WHERE project_id = $1
		ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []domain.Source
	for rows.Next() {
		source, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}

func (s *Store) GetSource(ctx context.Context, id string) (domain.Source, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, project_id, filename, source_type, status, artifact_path, content_hash, byte_size, created_at
		FROM sources
		WHERE id = $1
	`, id)
	return scanSource(row)
}

func (s *Store) CreateRunWithJob(ctx context.Context, source domain.Source, recipe string) (domain.Run, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Run{}, err
	}
	defer tx.Rollback(ctx)

	run, err := scanRun(tx.QueryRow(ctx, `
		INSERT INTO runs (project_id, source_id, recipe_snapshot, status, progress)
		VALUES ($1, $2, $3::jsonb, 'queued', 0)
		RETURNING id, project_id, COALESCE(source_id::text, ''), status, progress,
			total_samples, accepted_samples, rejected_samples, issue_count, error_message, created_at, updated_at
	`, source.ProjectID, source.ID, recipe))
	if err != nil {
		return domain.Run{}, err
	}

	payload, err := json.Marshal(map[string]string{
		"job_type":      "import_dataset",
		"project_id":    source.ProjectID,
		"run_id":        run.ID,
		"source_id":     source.ID,
		"artifact_path": source.ArtifactPath,
	})
	if err != nil {
		return domain.Run{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO jobs (project_id, run_id, job_type, payload)
		VALUES ($1, $2, 'import_dataset', $3::jsonb)
	`, source.ProjectID, run.ID, string(payload)); err != nil {
		return domain.Run{}, err
	}

	return run, tx.Commit(ctx)
}

func (s *Store) ListRuns(ctx context.Context, projectID string) ([]domain.Run, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, project_id, COALESCE(source_id::text, ''), status, progress,
			total_samples, accepted_samples, rejected_samples, issue_count, error_message, created_at, updated_at
		FROM runs
		WHERE project_id = $1
		ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []domain.Run
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (s *Store) ListSamples(ctx context.Context, projectID, status string, limit, offset int) ([]domain.Sample, error) {
	query := `
		SELECT id, project_id, COALESCE(run_id::text, ''), COALESCE(source_id::text, ''),
			status, input_text, output_text, quality_score, token_count, created_at
		FROM samples
		WHERE project_id = $1
	`
	args := []any{projectID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []domain.Sample
	for rows.Next() {
		sample, err := scanSample(rows)
		if err != nil {
			return nil, err
		}
		samples = append(samples, sample)
	}
	return samples, rows.Err()
}

func (s *Store) ReviewSample(ctx context.Context, id, status, reviewer, note string) (domain.Sample, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Sample{}, err
	}
	defer tx.Rollback(ctx)

	sample, err := scanSample(tx.QueryRow(ctx, `
		UPDATE samples
		SET status = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, project_id, COALESCE(run_id::text, ''), COALESCE(source_id::text, ''),
			status, input_text, output_text, quality_score, token_count, created_at
	`, id, status))
	if err != nil {
		return domain.Sample{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO review_records (sample_id, action, reviewer, note)
		VALUES ($1, $2, $3, $4)
	`, id, status, reviewer, note); err != nil {
		return domain.Sample{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Sample{}, err
	}
	return sample, nil
}

func (s *Store) ListQualityIssues(ctx context.Context, projectID string, limit int) ([]domain.QualityIssue, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, issue_type, severity, message, created_at
		FROM quality_issues
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []domain.QualityIssue
	for rows.Next() {
		var issue domain.QualityIssue
		if err := rows.Scan(&issue.ID, &issue.IssueType, &issue.Severity, &issue.Message, &issue.CreatedAt); err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}
	return issues, rows.Err()
}

func (s *Store) CreateDatasetVersionJob(ctx context.Context, projectID, runID, name, exportPath string) (domain.DatasetVersion, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.DatasetVersion{}, err
	}
	defer tx.Rollback(ctx)

	version, err := scanDatasetVersion(tx.QueryRow(ctx, `
		INSERT INTO dataset_versions (project_id, run_id, version_name, status, artifact_path)
		VALUES ($1, NULLIF($2, '')::uuid, $3, 'building', $4)
		RETURNING id, project_id, COALESCE(run_id::text, ''), version_name, status, artifact_path, sample_count, created_at
	`, projectID, runID, name, exportPath))
	if err != nil {
		return domain.DatasetVersion{}, err
	}

	payload, err := json.Marshal(map[string]string{
		"job_type":           "export_dataset",
		"project_id":         projectID,
		"run_id":             runID,
		"dataset_version_id": version.ID,
		"export_path":        exportPath,
	})
	if err != nil {
		return domain.DatasetVersion{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO jobs (project_id, run_id, job_type, payload)
		VALUES ($1, NULLIF($2, '')::uuid, 'export_dataset', $3::jsonb)
	`, projectID, runID, string(payload)); err != nil {
		return domain.DatasetVersion{}, err
	}
	return version, tx.Commit(ctx)
}

func (s *Store) ListDatasetVersions(ctx context.Context, projectID string) ([]domain.DatasetVersion, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, project_id, COALESCE(run_id::text, ''), version_name, status, artifact_path, sample_count, created_at
		FROM dataset_versions
		WHERE project_id = $1
		ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []domain.DatasetVersion
	for rows.Next() {
		version, err := scanDatasetVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (s *Store) ClaimJob(ctx context.Context, workerID string) (domain.Job, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE jobs
		SET status = 'claimed', claimed_by = $1, attempts = attempts + 1, heartbeat_at = now(), started_at = COALESCE(started_at, now()), updated_at = now()
		WHERE id = (
			SELECT id FROM jobs
			WHERE status = 'pending'
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, COALESCE(project_id::text, ''), COALESCE(run_id::text, ''),
			job_type, status, payload::text, attempts, max_attempts, error_message
	`, workerID)
	return scanJob(row)
}

func (s *Store) MarkJobRunning(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET status = 'running', heartbeat_at = now(), updated_at = now() WHERE id = $1`, id)
	return err
}

func (s *Store) HeartbeatJob(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET heartbeat_at = now(), updated_at = now() WHERE id = $1`, id)
	return err
}

func (s *Store) CompleteJob(ctx context.Context, id, result string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE jobs
		SET status = 'succeeded', result = $2::jsonb, finished_at = now(), updated_at = now()
		WHERE id = $1
	`, id, result)
	return err
}

func (s *Store) FailJob(ctx context.Context, id, message string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE jobs
		SET status = CASE WHEN attempts >= max_attempts THEN 'failed_final' ELSE 'pending' END,
			error_message = $2,
			finished_at = CASE WHEN attempts >= max_attempts THEN now() ELSE finished_at END,
			updated_at = now()
		WHERE id = $1
	`, id, message)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProject(row scanner) (domain.Project, error) {
	var project domain.Project
	err := row.Scan(&project.ID, &project.Name, &project.Description, &project.Domain, &project.CreatedAt, &project.UpdatedAt)
	return project, err
}

func scanSource(row scanner) (domain.Source, error) {
	var source domain.Source
	err := row.Scan(&source.ID, &source.ProjectID, &source.Filename, &source.SourceType, &source.Status, &source.ArtifactPath, &source.ContentHash, &source.ByteSize, &source.CreatedAt)
	return source, err
}

func scanRun(row scanner) (domain.Run, error) {
	var run domain.Run
	err := row.Scan(&run.ID, &run.ProjectID, &run.SourceID, &run.Status, &run.Progress, &run.TotalSamples, &run.AcceptedSamples, &run.RejectedSamples, &run.IssueCount, &run.ErrorMessage, &run.CreatedAt, &run.UpdatedAt)
	return run, err
}

func scanSample(row scanner) (domain.Sample, error) {
	var sample domain.Sample
	err := row.Scan(&sample.ID, &sample.ProjectID, &sample.RunID, &sample.SourceID, &sample.Status, &sample.InputText, &sample.OutputText, &sample.QualityScore, &sample.TokenCount, &sample.CreatedAt)
	return sample, err
}

func scanDatasetVersion(row scanner) (domain.DatasetVersion, error) {
	var version domain.DatasetVersion
	err := row.Scan(&version.ID, &version.ProjectID, &version.RunID, &version.VersionName, &version.Status, &version.ArtifactPath, &version.SampleCount, &version.CreatedAt)
	return version, err
}

func scanJob(row scanner) (domain.Job, error) {
	var job domain.Job
	err := row.Scan(&job.ID, &job.ProjectID, &job.RunID, &job.JobType, &job.Status, &job.Payload, &job.Attempts, &job.MaxAttempts, &job.ErrorMessage)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Job{}, ErrNoJob
	}
	return job, err
}

var ErrNoJob = errors.New("no pending job")
