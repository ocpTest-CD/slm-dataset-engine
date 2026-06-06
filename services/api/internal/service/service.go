package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/domain"
	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/repository"
	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/storage"
)

type App struct {
	store   *repository.Store
	storage *storage.Local
	logger  *slog.Logger
}

func New(store *repository.Store, localStorage *storage.Local, logger *slog.Logger) *App {
	return &App{store: store, storage: localStorage, logger: logger}
}

func (a *App) CreateProject(ctx context.Context, name, description, domainName string) (domain.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Project{}, errors.New("project name is required")
	}
	return a.store.CreateProject(ctx, name, description, domainName)
}

func (a *App) ListProjects(ctx context.Context) ([]domain.Project, error) {
	return a.store.ListProjects(ctx)
}

func (a *App) GetProject(ctx context.Context, id string) (domain.Project, error) {
	return a.store.GetProject(ctx, id)
}

func (a *App) UploadSource(ctx context.Context, projectID, filename string, reader io.Reader) (domain.Source, error) {
	if projectID == "" {
		return domain.Source{}, errors.New("project id is required")
	}
	if filename == "" {
		filename = "dataset.txt"
	}

	saved, err := a.storage.SaveSource(projectID, filename, reader)
	if err != nil {
		return domain.Source{}, err
	}

	sourceType := detectSourceType(filename)
	source := domain.Source{
		ProjectID:    projectID,
		Filename:     filepath.Base(filename),
		SourceType:   sourceType,
		ArtifactPath: saved.Path,
		ContentHash:  saved.Hash,
		ByteSize:     saved.Size,
	}
	return a.store.CreateSource(ctx, source)
}

func (a *App) ListSources(ctx context.Context, projectID string) ([]domain.Source, error) {
	return a.store.ListSources(ctx, projectID)
}

func (a *App) StartRun(ctx context.Context, sourceID string) (domain.Run, error) {
	source, err := a.store.GetSource(ctx, sourceID)
	if err != nil {
		return domain.Run{}, err
	}
	recipe := `{"name":"default_import","steps":["parse","normalize","quality"]}`
	return a.store.CreateRunWithJob(ctx, source, recipe)
}

func (a *App) ListRuns(ctx context.Context, projectID string) ([]domain.Run, error) {
	return a.store.ListRuns(ctx, projectID)
}

func (a *App) ListSamples(ctx context.Context, projectID, status string, limit, offset int) ([]domain.Sample, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return a.store.ListSamples(ctx, projectID, status, limit, offset)
}

func (a *App) ReviewSample(ctx context.Context, id, status, reviewer, note string) (domain.Sample, error) {
	if status != "accepted" && status != "rejected" && status != "pending_review" {
		return domain.Sample{}, errors.New("invalid review status")
	}
	return a.store.ReviewSample(ctx, id, status, reviewer, note)
}

func (a *App) ListQualityIssues(ctx context.Context, projectID string, limit int) ([]domain.QualityIssue, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return a.store.ListQualityIssues(ctx, projectID, limit)
}

func (a *App) CreateDatasetVersion(ctx context.Context, projectID, runID, name string) (domain.DatasetVersion, error) {
	if strings.TrimSpace(name) == "" {
		name = "dataset-version"
	}
	path := a.storage.ExportPath(projectID, "pending")
	version, err := a.store.CreateDatasetVersionJob(ctx, projectID, runID, name, path)
	if err != nil {
		return domain.DatasetVersion{}, err
	}
	return version, nil
}

func (a *App) ListDatasetVersions(ctx context.Context, projectID string) ([]domain.DatasetVersion, error) {
	return a.store.ListDatasetVersions(ctx, projectID)
}

func (a *App) ClaimJob(ctx context.Context, workerID string) (domain.Job, error) {
	if workerID == "" {
		workerID = "worker"
	}
	return a.store.ClaimJob(ctx, workerID)
}

func (a *App) MarkJobRunning(ctx context.Context, id string) error {
	return a.store.MarkJobRunning(ctx, id)
}

func (a *App) HeartbeatJob(ctx context.Context, id string) error {
	return a.store.HeartbeatJob(ctx, id)
}

func (a *App) CompleteJob(ctx context.Context, id, result string) error {
	return a.store.CompleteJob(ctx, id, result)
}

func (a *App) FailJob(ctx context.Context, id, message string) error {
	return a.store.FailJob(ctx, id, message)
}

func detectSourceType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jsonl":
		return "jsonl"
	case ".csv":
		return "csv"
	case ".md", ".markdown":
		return "markdown"
	default:
		return "text"
	}
}
