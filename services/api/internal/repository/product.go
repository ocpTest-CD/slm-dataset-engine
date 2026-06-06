package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/domain"
)

const (
	DefaultUserID      = "00000000-0000-0000-0000-000000000001"
	DefaultWorkspaceID = "00000000-0000-0000-0000-000000000101"
)

func TokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Store) ResolvePrincipal(ctx context.Context, token string) (domain.Principal, error) {
	if token == "" {
		return domain.Principal{UserID: DefaultUserID, WorkspaceID: DefaultWorkspaceID, Role: "owner"}, nil
	}
	row := s.pool.QueryRow(ctx, `
		SELECT user_id, workspace_id, role
		FROM api_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL
			AND (expires_at IS NULL OR expires_at > now())
	`, TokenHash(token))
	var principal domain.Principal
	if err := row.Scan(&principal.UserID, &principal.WorkspaceID, &principal.Role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Principal{}, errors.New("invalid api token")
		}
		return domain.Principal{}, err
	}
	return principal, nil
}

func (s *Store) GetMe(ctx context.Context, principal domain.Principal) (domain.Me, error) {
	user, err := scanUser(s.pool.QueryRow(ctx, `SELECT id, email, name, created_at FROM users WHERE id = $1`, principal.UserID))
	if err != nil {
		return domain.Me{}, err
	}
	workspaces, err := s.ListWorkspaces(ctx, principal.UserID)
	if err != nil {
		return domain.Me{}, err
	}
	var current domain.Workspace
	for _, workspace := range workspaces {
		if workspace.ID == principal.WorkspaceID {
			current = workspace
			current.Role = principal.Role
			break
		}
	}
	return domain.Me{User: user, Workspace: current, Principal: principal, Workspaces: workspaces}, nil
}

func (s *Store) ListWorkspaces(ctx context.Context, userID string) ([]domain.Workspace, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT w.id, w.name, wm.role, w.created_at
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id = w.id
		WHERE wm.user_id = $1
		ORDER BY w.created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workspaces := make([]domain.Workspace, 0)
	for rows.Next() {
		var workspace domain.Workspace
		if err := rows.Scan(&workspace.ID, &workspace.Name, &workspace.Role, &workspace.CreatedAt); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, workspace)
	}
	return workspaces, rows.Err()
}

func (s *Store) CreateMCPServer(ctx context.Context, workspaceID, projectID, name, endpoint, transport string) (domain.MCPServer, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO mcp_servers (workspace_id, project_id, name, endpoint, transport, status)
		VALUES ($1, $2, $3, $4, $5, 'registered')
		RETURNING id, workspace_id, project_id, name, endpoint, transport, status, config::text, created_at, updated_at
	`, workspaceID, projectID, name, endpoint, transport)
	return scanMCPServer(row)
}

func (s *Store) ListMCPServers(ctx context.Context, workspaceID, projectID string) ([]domain.MCPServer, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, project_id, name, endpoint, transport, status, config::text, created_at, updated_at
		FROM mcp_servers
		WHERE workspace_id = $1 AND project_id = $2
		ORDER BY created_at DESC
	`, workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	servers := make([]domain.MCPServer, 0)
	for rows.Next() {
		server, err := scanMCPServer(rows)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, rows.Err()
}

func (s *Store) CreateMCPTool(ctx context.Context, workspaceID, serverID, name, description, inputSchema, outputSchema string) (domain.MCPTool, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO mcp_tools (workspace_id, server_id, name, description, input_schema, output_schema, enabled)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, true)
		RETURNING id, workspace_id, server_id, name, description, input_schema::text, output_schema::text, enabled, created_at, updated_at
	`, workspaceID, serverID, name, description, jsonOrEmpty(inputSchema), jsonOrEmpty(outputSchema))
	return scanMCPTool(row)
}

func (s *Store) ListMCPTools(ctx context.Context, workspaceID, serverID string) ([]domain.MCPTool, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, server_id, name, description, input_schema::text, output_schema::text, enabled, created_at, updated_at
		FROM mcp_tools
		WHERE workspace_id = $1 AND server_id = $2
		ORDER BY created_at DESC
	`, workspaceID, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tools := make([]domain.MCPTool, 0)
	for rows.Next() {
		tool, err := scanMCPTool(rows)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, rows.Err()
}

func (s *Store) GetMCPToolContext(ctx context.Context, workspaceID, toolID string) (domain.MCPTool, domain.MCPServer, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT t.id, t.workspace_id, t.server_id, t.name, t.description, t.input_schema::text, t.output_schema::text, t.enabled, t.created_at, t.updated_at,
			s.id, s.workspace_id, s.project_id, s.name, s.endpoint, s.transport, s.status, s.config::text, s.created_at, s.updated_at
		FROM mcp_tools t
		JOIN mcp_servers s ON s.id = t.server_id
		WHERE t.workspace_id = $1 AND t.id = $2
	`, workspaceID, toolID)
	var tool domain.MCPTool
	var server domain.MCPServer
	err := row.Scan(
		&tool.ID, &tool.WorkspaceID, &tool.ServerID, &tool.Name, &tool.Description, &tool.InputSchema, &tool.OutputSchema, &tool.Enabled, &tool.CreatedAt, &tool.UpdatedAt,
		&server.ID, &server.WorkspaceID, &server.ProjectID, &server.Name, &server.Endpoint, &server.Transport, &server.Status, &server.Config, &server.CreatedAt, &server.UpdatedAt,
	)
	return tool, server, err
}

func (s *Store) CreateToolInvocation(ctx context.Context, principal domain.Principal, tool domain.MCPTool, server domain.MCPServer, input, artifactPath string) (domain.ToolInvocation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ToolInvocation{}, err
	}
	defer tx.Rollback(ctx)

	invocation, err := scanToolInvocation(tx.QueryRow(ctx, `
		INSERT INTO tool_invocations (workspace_id, project_id, server_id, tool_id, user_id, status, input)
		VALUES ($1, $2, $3, $4, $5, 'queued', $6::jsonb)
		RETURNING id, workspace_id, project_id, COALESCE(server_id::text, ''), COALESCE(tool_id::text, ''), COALESCE(user_id::text, ''),
			status, input::text, output::text, error_message, duration_ms, COALESCE(job_id::text, ''), $7::text, created_at, updated_at
	`, principal.WorkspaceID, server.ProjectID, server.ID, tool.ID, principal.UserID, jsonOrEmpty(input), tool.Name))
	if err != nil {
		return domain.ToolInvocation{}, err
	}

	payloadMap := map[string]string{
		"job_type":      "mcp_tool_invocation",
		"workspace_id":  principal.WorkspaceID,
		"project_id":    server.ProjectID,
		"server_id":     server.ID,
		"tool_id":       tool.ID,
		"tool_name":     tool.Name,
		"invocation_id": invocation.ID,
		"input":         jsonOrEmpty(input),
		"artifact_path": artifactPath,
	}
	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return domain.ToolInvocation{}, err
	}
	job, err := scanJob(tx.QueryRow(ctx, `
		INSERT INTO jobs (workspace_id, project_id, invocation_id, job_type, payload, stage, progress, message)
		VALUES ($1, $2, $3, 'mcp_tool_invocation', $4::jsonb, 'queued', 0, '等待 Worker 执行')
		RETURNING id, workspace_id, COALESCE(project_id::text, ''), COALESCE(run_id::text, ''), COALESCE(invocation_id::text, ''),
			job_type, status, stage, progress, message, payload::text, attempts, max_attempts, error_message
	`, principal.WorkspaceID, server.ProjectID, invocation.ID, string(payload)))
	if err != nil {
		return domain.ToolInvocation{}, err
	}
	payloadMap["job_id"] = job.ID
	payload, err = json.Marshal(payloadMap)
	if err != nil {
		return domain.ToolInvocation{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE tool_invocations SET job_id = $2, updated_at = now() WHERE id = $1`, invocation.ID, job.ID); err != nil {
		return domain.ToolInvocation{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE jobs SET payload = $2::jsonb WHERE id = $1`, job.ID, string(payload)); err != nil {
		return domain.ToolInvocation{}, err
	}
	invocation.JobID = job.ID
	return invocation, tx.Commit(ctx)
}

func (s *Store) ListToolInvocations(ctx context.Context, workspaceID, projectID string) ([]domain.ToolInvocation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.workspace_id, i.project_id, COALESCE(i.server_id::text, ''), COALESCE(i.tool_id::text, ''), COALESCE(i.user_id::text, ''),
			i.status, i.input::text, i.output::text, i.error_message, i.duration_ms, COALESCE(i.job_id::text, ''), COALESCE(t.name, ''), i.created_at, i.updated_at
		FROM tool_invocations i
		LEFT JOIN mcp_tools t ON t.id = i.tool_id
		WHERE i.workspace_id = $1 AND i.project_id = $2
		ORDER BY i.created_at DESC
		LIMIT 80
	`, workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.ToolInvocation, 0)
	for rows.Next() {
		item, err := scanToolInvocation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListProjectJobs(ctx context.Context, workspaceID, projectID string) ([]domain.Job, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, COALESCE(project_id::text, ''), COALESCE(run_id::text, ''), COALESCE(invocation_id::text, ''),
			job_type, status, stage, progress, message, payload::text, attempts, max_attempts, error_message
		FROM jobs
		WHERE workspace_id = $1 AND project_id = $2
		ORDER BY created_at DESC
		LIMIT 80
	`, workspaceID, projectID)
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

	var workspaceID string
	if err := tx.QueryRow(ctx, `
		UPDATE jobs
		SET stage = $2, progress = $3, message = $4, heartbeat_at = now(), updated_at = now()
		WHERE id = $1
		RETURNING workspace_id
	`, jobID, stage, progress, message).Scan(&workspaceID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO job_events (workspace_id, job_id, event_type, stage, progress, message, metadata)
		VALUES ($1, $2, 'progress', $3, $4, $5, $6::jsonb)
	`, workspaceID, jobID, stage, progress, message, jsonOrEmpty(metadata)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ListJobEvents(ctx context.Context, workspaceID, jobID string) ([]domain.JobEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, job_id, event_type, stage, progress, message, metadata::text, created_at
		FROM job_events
		WHERE workspace_id = $1 AND job_id = $2
		ORDER BY created_at DESC
		LIMIT 100
	`, workspaceID, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]domain.JobEvent, 0)
	for rows.Next() {
		var event domain.JobEvent
		if err := rows.Scan(&event.ID, &event.WorkspaceID, &event.JobID, &event.EventType, &event.Stage, &event.Progress, &event.Message, &event.Metadata, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) ListArtifacts(ctx context.Context, workspaceID, projectID string) ([]domain.Artifact, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, project_id, COALESCE(invocation_id::text, ''), COALESCE(job_id::text, ''),
			name, artifact_type, status, manifest::text, created_at, updated_at
		FROM artifacts
		WHERE workspace_id = $1 AND project_id = $2
		ORDER BY created_at DESC
		LIMIT 80
	`, workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	artifacts := make([]domain.Artifact, 0)
	for rows.Next() {
		artifact, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, rows.Err()
}

func (s *Store) ListArtifactFiles(ctx context.Context, workspaceID, artifactID string) ([]domain.ArtifactFile, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, artifact_id, file_name, file_path, mime_type, byte_size, sha256, created_at
		FROM artifact_files
		WHERE workspace_id = $1 AND artifact_id = $2
		ORDER BY created_at
	`, workspaceID, artifactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]domain.ArtifactFile, 0)
	for rows.Next() {
		file, err := scanArtifactFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

func (s *Store) GetArtifactFile(ctx context.Context, workspaceID, fileID string) (domain.ArtifactFile, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, workspace_id, artifact_id, file_name, file_path, mime_type, byte_size, sha256, created_at
		FROM artifact_files
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, fileID)
	return scanArtifactFile(row)
}

func (s *Store) CreateAuditLog(ctx context.Context, workspaceID, projectID, userID, action, targetType, targetID, metadata string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_logs (workspace_id, project_id, user_id, action, target_type, target_id, metadata)
		VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4, $5, $6, $7::jsonb)
	`, workspaceID, projectID, userID, action, targetType, targetID, jsonOrEmpty(metadata))
	return err
}

func (s *Store) ListAuditLogs(ctx context.Context, workspaceID, projectID string) ([]domain.AuditLog, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, COALESCE(project_id::text, ''), COALESCE(user_id::text, ''),
			action, target_type, target_id, metadata::text, created_at
		FROM audit_logs
		WHERE workspace_id = $1 AND project_id = $2
		ORDER BY created_at DESC
		LIMIT 100
	`, workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]domain.AuditLog, 0)
	for rows.Next() {
		var log domain.AuditLog
		if err := rows.Scan(&log.ID, &log.WorkspaceID, &log.ProjectID, &log.UserID, &log.Action, &log.TargetType, &log.TargetID, &log.Metadata, &log.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func jsonOrEmpty(value string) string {
	var decoded any
	if value == "" || json.Unmarshal([]byte(value), &decoded) != nil {
		return "{}"
	}
	return value
}

func scanUser(row scanner) (domain.User, error) {
	var user domain.User
	err := row.Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
	return user, err
}

func scanMCPServer(row scanner) (domain.MCPServer, error) {
	var server domain.MCPServer
	err := row.Scan(&server.ID, &server.WorkspaceID, &server.ProjectID, &server.Name, &server.Endpoint, &server.Transport, &server.Status, &server.Config, &server.CreatedAt, &server.UpdatedAt)
	return server, err
}

func scanMCPTool(row scanner) (domain.MCPTool, error) {
	var tool domain.MCPTool
	err := row.Scan(&tool.ID, &tool.WorkspaceID, &tool.ServerID, &tool.Name, &tool.Description, &tool.InputSchema, &tool.OutputSchema, &tool.Enabled, &tool.CreatedAt, &tool.UpdatedAt)
	return tool, err
}

func scanToolInvocation(row scanner) (domain.ToolInvocation, error) {
	var invocation domain.ToolInvocation
	err := row.Scan(&invocation.ID, &invocation.WorkspaceID, &invocation.ProjectID, &invocation.ServerID, &invocation.ToolID, &invocation.UserID, &invocation.Status, &invocation.Input, &invocation.Output, &invocation.ErrorMessage, &invocation.DurationMS, &invocation.JobID, &invocation.ToolName, &invocation.CreatedAt, &invocation.UpdatedAt)
	return invocation, err
}

func scanArtifact(row scanner) (domain.Artifact, error) {
	var artifact domain.Artifact
	err := row.Scan(&artifact.ID, &artifact.WorkspaceID, &artifact.ProjectID, &artifact.InvocationID, &artifact.JobID, &artifact.Name, &artifact.ArtifactType, &artifact.Status, &artifact.Manifest, &artifact.CreatedAt, &artifact.UpdatedAt)
	return artifact, err
}

func scanArtifactFile(row scanner) (domain.ArtifactFile, error) {
	var file domain.ArtifactFile
	err := row.Scan(&file.ID, &file.WorkspaceID, &file.ArtifactID, &file.FileName, &file.FilePath, &file.MimeType, &file.ByteSize, &file.SHA256, &file.CreatedAt)
	return file, err
}
