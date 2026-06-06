package domain

import "time"

type Project struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
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
	WorkspaceID  string `json:"workspace_id"`
	ProjectID    string `json:"project_id"`
	RunID        string `json:"run_id"`
	InvocationID string `json:"invocation_id"`
	JobType      string `json:"job_type"`
	Status       string `json:"status"`
	Stage        string `json:"stage"`
	Progress     int    `json:"progress"`
	Message      string `json:"message"`
	Payload      string `json:"payload"`
	Attempts     int    `json:"attempts"`
	MaxAttempts  int    `json:"max_attempts"`
	ErrorMessage string `json:"error_message"`
}

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Principal struct {
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
	Role        string `json:"role"`
}

type Me struct {
	User       User        `json:"user"`
	Workspace  Workspace   `json:"workspace"`
	Principal  Principal   `json:"principal"`
	Workspaces []Workspace `json:"workspaces"`
}

type MCPServer struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Endpoint    string    `json:"endpoint"`
	Transport   string    `json:"transport"`
	Status      string    `json:"status"`
	Config      string    `json:"config"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MCPTool struct {
	ID           string    `json:"id"`
	WorkspaceID  string    `json:"workspace_id"`
	ServerID     string    `json:"server_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	InputSchema  string    `json:"input_schema"`
	OutputSchema string    `json:"output_schema"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ToolInvocation struct {
	ID           string    `json:"id"`
	WorkspaceID  string    `json:"workspace_id"`
	ProjectID    string    `json:"project_id"`
	ServerID     string    `json:"server_id"`
	ToolID       string    `json:"tool_id"`
	UserID       string    `json:"user_id"`
	Status       string    `json:"status"`
	Input        string    `json:"input"`
	Output       string    `json:"output"`
	ErrorMessage string    `json:"error_message"`
	DurationMS   int       `json:"duration_ms"`
	JobID        string    `json:"job_id"`
	ToolName     string    `json:"tool_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type JobEvent struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	JobID       string    `json:"job_id"`
	EventType   string    `json:"event_type"`
	Stage       string    `json:"stage"`
	Progress    int       `json:"progress"`
	Message     string    `json:"message"`
	Metadata    string    `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
}

type Artifact struct {
	ID           string    `json:"id"`
	WorkspaceID  string    `json:"workspace_id"`
	ProjectID    string    `json:"project_id"`
	InvocationID string    `json:"invocation_id"`
	JobID        string    `json:"job_id"`
	Name         string    `json:"name"`
	ArtifactType string    `json:"artifact_type"`
	Status       string    `json:"status"`
	Manifest     string    `json:"manifest"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ArtifactFile struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	ArtifactID  string    `json:"artifact_id"`
	FileName    string    `json:"file_name"`
	FilePath    string    `json:"file_path"`
	MimeType    string    `json:"mime_type"`
	ByteSize    int64     `json:"byte_size"`
	SHA256      string    `json:"sha256"`
	CreatedAt   time.Time `json:"created_at"`
}

type AuditLog struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	ProjectID   string    `json:"project_id"`
	UserID      string    `json:"user_id"`
	Action      string    `json:"action"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	Metadata    string    `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
}
