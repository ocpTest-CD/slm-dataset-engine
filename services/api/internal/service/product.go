package service

import (
	"context"
	"errors"
	"strings"

	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/domain"
)

func (a *App) ResolvePrincipal(ctx context.Context, token string) (domain.Principal, error) {
	return a.store.ResolvePrincipal(ctx, token)
}

func (a *App) GetMe(ctx context.Context, principal domain.Principal) (domain.Me, error) {
	return a.store.GetMe(ctx, principal)
}

func (a *App) ListWorkspaces(ctx context.Context, principal domain.Principal) ([]domain.Workspace, error) {
	return a.store.ListWorkspaces(ctx, principal.UserID)
}

func (a *App) CreateMCPServer(ctx context.Context, principal domain.Principal, projectID, name, endpoint, transport string) (domain.MCPServer, error) {
	if !canWrite(principal.Role) {
		return domain.MCPServer{}, errors.New("permission denied")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Local MCP Server"
	}
	if transport == "" {
		transport = "stdio"
	}
	server, err := a.store.CreateMCPServer(ctx, principal.WorkspaceID, projectID, name, endpoint, transport)
	if err != nil {
		return domain.MCPServer{}, err
	}
	if _, err := a.store.CreateMCPTool(
		ctx,
		principal.WorkspaceID,
		server.ID,
		"echo_artifact",
		"将输入内容写成可下载 Artifact，用于验证 MCP 产品化闭环。",
		`{"type":"object","properties":{"text":{"type":"string"},"metadata":{"type":"object"}}}`,
		`{"type":"object","properties":{"artifact_id":{"type":"string"},"file_count":{"type":"integer"}}}`,
	); err != nil {
		return domain.MCPServer{}, err
	}
	_ = a.store.CreateAuditLog(ctx, principal.WorkspaceID, projectID, principal.UserID, "mcp_server.created", "mcp_server", server.ID, "{}")
	return server, nil
}

func (a *App) ListMCPServers(ctx context.Context, principal domain.Principal, projectID string) ([]domain.MCPServer, error) {
	return a.store.ListMCPServers(ctx, principal.WorkspaceID, projectID)
}

func (a *App) CreateMCPTool(ctx context.Context, principal domain.Principal, serverID, name, description, inputSchema, outputSchema string) (domain.MCPTool, error) {
	if !canWrite(principal.Role) {
		return domain.MCPTool{}, errors.New("permission denied")
	}
	if strings.TrimSpace(name) == "" {
		return domain.MCPTool{}, errors.New("tool name is required")
	}
	tool, err := a.store.CreateMCPTool(ctx, principal.WorkspaceID, serverID, name, description, inputSchema, outputSchema)
	if err != nil {
		return domain.MCPTool{}, err
	}
	_ = a.store.CreateAuditLog(ctx, principal.WorkspaceID, "", principal.UserID, "mcp_tool.created", "mcp_tool", tool.ID, "{}")
	return tool, nil
}

func (a *App) ListMCPTools(ctx context.Context, principal domain.Principal, serverID string) ([]domain.MCPTool, error) {
	return a.store.ListMCPTools(ctx, principal.WorkspaceID, serverID)
}

func (a *App) InvokeMCPTool(ctx context.Context, principal domain.Principal, toolID, input string) (domain.ToolInvocation, error) {
	if !canWrite(principal.Role) {
		return domain.ToolInvocation{}, errors.New("permission denied")
	}
	tool, server, err := a.store.GetMCPToolContext(ctx, principal.WorkspaceID, toolID)
	if err != nil {
		return domain.ToolInvocation{}, err
	}
	if !tool.Enabled {
		return domain.ToolInvocation{}, errors.New("tool is disabled")
	}
	path := a.storage.InvocationArtifactPath(server.ProjectID, "pending")
	invocation, err := a.store.CreateToolInvocation(ctx, principal, tool, server, input, path)
	if err != nil {
		return domain.ToolInvocation{}, err
	}
	_ = a.store.CreateAuditLog(ctx, principal.WorkspaceID, server.ProjectID, principal.UserID, "mcp_tool.invoked", "tool_invocation", invocation.ID, input)
	return invocation, nil
}

func (a *App) ListToolInvocations(ctx context.Context, principal domain.Principal, projectID string) ([]domain.ToolInvocation, error) {
	return a.store.ListToolInvocations(ctx, principal.WorkspaceID, projectID)
}

func (a *App) ListProjectJobs(ctx context.Context, principal domain.Principal, projectID string) ([]domain.Job, error) {
	return a.store.ListProjectJobs(ctx, principal.WorkspaceID, projectID)
}

func (a *App) UpdateJobProgress(ctx context.Context, jobID, stage string, progress int, message, metadata string) error {
	return a.store.UpdateJobProgress(ctx, jobID, stage, progress, message, metadata)
}

func (a *App) ListJobEvents(ctx context.Context, principal domain.Principal, jobID string) ([]domain.JobEvent, error) {
	return a.store.ListJobEvents(ctx, principal.WorkspaceID, jobID)
}

func (a *App) ListArtifacts(ctx context.Context, principal domain.Principal, projectID string) ([]domain.Artifact, error) {
	return a.store.ListArtifacts(ctx, principal.WorkspaceID, projectID)
}

func (a *App) ListArtifactFiles(ctx context.Context, principal domain.Principal, artifactID string) ([]domain.ArtifactFile, error) {
	return a.store.ListArtifactFiles(ctx, principal.WorkspaceID, artifactID)
}

func (a *App) GetArtifactFile(ctx context.Context, principal domain.Principal, fileID string) (domain.ArtifactFile, error) {
	file, err := a.store.GetArtifactFile(ctx, principal.WorkspaceID, fileID)
	if err == nil {
		_ = a.store.CreateAuditLog(ctx, principal.WorkspaceID, "", principal.UserID, "artifact_file.downloaded", "artifact_file", file.ID, "{}")
	}
	return file, err
}

func (a *App) ListAuditLogs(ctx context.Context, principal domain.Principal, projectID string) ([]domain.AuditLog, error) {
	return a.store.ListAuditLogs(ctx, principal.WorkspaceID, projectID)
}

func canWrite(role string) bool {
	return role == "owner" || role == "member"
}
