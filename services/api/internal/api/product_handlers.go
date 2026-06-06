package api

import (
	"net/http"
	"os"
	"strings"

	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/domain"
)

func (r *Router) getMe(w http.ResponseWriter, req *http.Request) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	me, err := r.app.GetMe(req.Context(), principal)
	writeResult(w, me, err)
}

func (r *Router) listWorkspaces(w http.ResponseWriter, req *http.Request) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	workspaces, err := r.app.ListWorkspaces(req.Context(), principal)
	writeResult(w, workspaces, err)
}

func (r *Router) createMCPServer(w http.ResponseWriter, req *http.Request, projectID string) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	var body struct {
		Name      string `json:"name"`
		Endpoint  string `json:"endpoint"`
		Transport string `json:"transport"`
	}
	if err := readJSON(req, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	server, err := r.app.CreateMCPServer(req.Context(), principal, projectID, body.Name, body.Endpoint, body.Transport)
	writeResult(w, server, err)
}

func (r *Router) listMCPServers(w http.ResponseWriter, req *http.Request, projectID string) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	servers, err := r.app.ListMCPServers(req.Context(), principal, projectID)
	writeResult(w, servers, err)
}

func (r *Router) createMCPTool(w http.ResponseWriter, req *http.Request, serverID string) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	var body struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		InputSchema  string `json:"input_schema"`
		OutputSchema string `json:"output_schema"`
	}
	if err := readJSON(req, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tool, err := r.app.CreateMCPTool(req.Context(), principal, serverID, body.Name, body.Description, body.InputSchema, body.OutputSchema)
	writeResult(w, tool, err)
}

func (r *Router) listMCPTools(w http.ResponseWriter, req *http.Request, serverID string) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	tools, err := r.app.ListMCPTools(req.Context(), principal, serverID)
	writeResult(w, tools, err)
}

func (r *Router) invokeMCPTool(w http.ResponseWriter, req *http.Request, toolID string) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	var body struct {
		Input string `json:"input"`
	}
	if err := readJSON(req, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	invocation, err := r.app.InvokeMCPTool(req.Context(), principal, toolID, body.Input)
	writeResult(w, invocation, err)
}

func (r *Router) listInvocations(w http.ResponseWriter, req *http.Request, projectID string) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	invocations, err := r.app.ListToolInvocations(req.Context(), principal, projectID)
	writeResult(w, invocations, err)
}

func (r *Router) listProjectJobs(w http.ResponseWriter, req *http.Request, projectID string) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	jobs, err := r.app.ListProjectJobs(req.Context(), principal, projectID)
	writeResult(w, jobs, err)
}

func (r *Router) listJobEvents(w http.ResponseWriter, req *http.Request, jobID string) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	events, err := r.app.ListJobEvents(req.Context(), principal, jobID)
	writeResult(w, events, err)
}

func (r *Router) listArtifacts(w http.ResponseWriter, req *http.Request, projectID string) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	artifacts, err := r.app.ListArtifacts(req.Context(), principal, projectID)
	writeResult(w, artifacts, err)
}

func (r *Router) listArtifactFiles(w http.ResponseWriter, req *http.Request, artifactID string) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	files, err := r.app.ListArtifactFiles(req.Context(), principal, artifactID)
	writeResult(w, files, err)
}

func (r *Router) downloadArtifactFile(w http.ResponseWriter, req *http.Request, fileID string) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	file, err := r.app.GetArtifactFile(req.Context(), principal, fileID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if _, err := os.Stat(file.FilePath); err != nil {
		writeError(w, http.StatusNotFound, "artifact file not found")
		return
	}
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+file.FileName+`"`)
	http.ServeFile(w, req, file.FilePath)
}

func (r *Router) listAuditLogs(w http.ResponseWriter, req *http.Request, projectID string) {
	principal, err := r.principal(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	logs, err := r.app.ListAuditLogs(req.Context(), principal, projectID)
	writeResult(w, logs, err)
}

func (r *Router) principal(req *http.Request) (domain.Principal, error) {
	token := strings.TrimSpace(req.Header.Get("Authorization"))
	token = strings.TrimPrefix(token, "Bearer ")
	return r.app.ResolvePrincipal(req.Context(), token)
}
