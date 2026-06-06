package domain

import "time"

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Domain      string    `json:"domain"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Source struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Filename     string    `json:"filename"`
	SourceType   string    `json:"source_type"`
	Status       string    `json:"status"`
	ArtifactPath string    `json:"artifact_path"`
	ContentHash  string    `json:"content_hash"`
	ByteSize     int64     `json:"byte_size"`
	CreatedAt    time.Time `json:"created_at"`
}

type Run struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"project_id"`
	SourceID        string    `json:"source_id"`
	Status          string    `json:"status"`
	Progress        int       `json:"progress"`
	TotalSamples    int       `json:"total_samples"`
	AcceptedSamples int       `json:"accepted_samples"`
	RejectedSamples int       `json:"rejected_samples"`
	IssueCount      int       `json:"issue_count"`
	ErrorMessage    string    `json:"error_message"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Sample struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	RunID        string    `json:"run_id"`
	SourceID     string    `json:"source_id"`
	Status       string    `json:"status"`
	InputText    string    `json:"input_text"`
	OutputText   string    `json:"output_text"`
	QualityScore *float64  `json:"quality_score"`
	TokenCount   int       `json:"token_count"`
	CreatedAt    time.Time `json:"created_at"`
}

type QualityIssue struct {
	ID        string    `json:"id"`
	IssueType string    `json:"issue_type"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type DatasetVersion struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	RunID        string    `json:"run_id"`
	VersionName  string    `json:"version_name"`
	Status       string    `json:"status"`
	ArtifactPath string    `json:"artifact_path"`
	SampleCount  int       `json:"sample_count"`
	CreatedAt    time.Time `json:"created_at"`
}

type Job struct {
	ID           string `json:"id"`
	ProjectID    string `json:"project_id"`
	RunID        string `json:"run_id"`
	JobType      string `json:"job_type"`
	Status       string `json:"status"`
	Payload      string `json:"payload"`
	Attempts     int    `json:"attempts"`
	MaxAttempts  int    `json:"max_attempts"`
	ErrorMessage string `json:"error_message"`
}
