package service

import (
	"context"
	"errors"
	"strings"

	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/domain"
)

func (a *App) ListProjectJobs(ctx context.Context, projectID string) ([]domain.Job, error) {
	return a.store.ListProjectJobs(ctx, projectID)
}

func (a *App) UpdateJobProgress(ctx context.Context, jobID, stage string, progress int, message, metadata string) error {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	return a.store.UpdateJobProgress(ctx, jobID, stage, progress, message, metadata)
}

func (a *App) ListJobEvents(ctx context.Context, jobID string) ([]domain.JobEvent, error) {
	return a.store.ListJobEvents(ctx, jobID)
}

func (a *App) EditSample(ctx context.Context, id, inputText, outputText, editedBy, reason, nextStatus string) (domain.Sample, error) {
	if strings.TrimSpace(inputText) == "" {
		return domain.Sample{}, errors.New("input text is required")
	}
	if editedBy == "" {
		editedBy = "workbench"
	}
	if nextStatus == "" {
		nextStatus = "edited"
	}
	if nextStatus != "edited" && nextStatus != "pending_review" && nextStatus != "accepted" && nextStatus != "rejected" {
		return domain.Sample{}, errors.New("invalid sample status")
	}
	return a.store.EditSample(ctx, id, inputText, outputText, editedBy, reason, nextStatus)
}

func (a *App) ListSampleVersions(ctx context.Context, sampleID string) ([]domain.SampleVersion, error) {
	return a.store.ListSampleVersions(ctx, sampleID)
}

func (a *App) ListDatasetVersionFiles(ctx context.Context, versionID string) ([]domain.DatasetVersionFile, error) {
	return a.store.ListDatasetVersionFiles(ctx, versionID)
}

func (a *App) GetDatasetVersionFile(ctx context.Context, fileID string) (domain.DatasetVersionFile, error) {
	return a.store.GetDatasetVersionFile(ctx, fileID)
}
