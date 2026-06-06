package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/repository"
	"github.com/ocptest-cd/slm-dataset-engine/services/api/internal/service"
)

type Router struct {
	app    *service.App
	logger *slog.Logger
}

func NewRouter(app *service.App, logger *slog.Logger) http.Handler {
	router := &Router{app: app, logger: logger}
	return withCORS(router)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/healthz" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if !strings.HasPrefix(req.URL.Path, "/api/") {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	parts := splitPath(req.URL.Path)
	switch {
	case req.Method == http.MethodGet && match(parts, "api", "projects"):
		r.listProjects(w, req)
	case req.Method == http.MethodPost && match(parts, "api", "projects"):
		r.createProject(w, req)
	case req.Method == http.MethodGet && len(parts) == 3 && parts[0] == "api" && parts[1] == "projects":
		r.getProject(w, req, parts[2])
	case req.Method == http.MethodGet && len(parts) == 4 && parts[0] == "api" && parts[1] == "projects" && parts[3] == "sources":
		r.listSources(w, req, parts[2])
	case req.Method == http.MethodPost && len(parts) == 4 && parts[0] == "api" && parts[1] == "projects" && parts[3] == "sources":
		r.uploadSource(w, req, parts[2])
	case req.Method == http.MethodGet && len(parts) == 4 && parts[0] == "api" && parts[1] == "projects" && parts[3] == "runs":
		r.listRuns(w, req, parts[2])
	case req.Method == http.MethodGet && len(parts) == 4 && parts[0] == "api" && parts[1] == "projects" && parts[3] == "jobs":
		r.listProjectJobs(w, req, parts[2])
	case req.Method == http.MethodPost && len(parts) == 4 && parts[0] == "api" && parts[1] == "sources" && parts[3] == "runs":
		r.startRun(w, req, parts[2])
	case req.Method == http.MethodGet && len(parts) == 4 && parts[0] == "api" && parts[1] == "projects" && parts[3] == "samples":
		r.listSamples(w, req, parts[2])
	case req.Method == http.MethodPatch && len(parts) == 4 && parts[0] == "api" && parts[1] == "samples" && parts[3] == "edit":
		r.editSample(w, req, parts[2])
	case req.Method == http.MethodGet && len(parts) == 4 && parts[0] == "api" && parts[1] == "samples" && parts[3] == "versions":
		r.listSampleVersions(w, req, parts[2])
	case req.Method == http.MethodPatch && len(parts) == 4 && parts[0] == "api" && parts[1] == "samples" && parts[3] == "review":
		r.reviewSample(w, req, parts[2])
	case req.Method == http.MethodGet && len(parts) == 4 && parts[0] == "api" && parts[1] == "projects" && parts[3] == "quality-issues":
		r.listQualityIssues(w, req, parts[2])
	case req.Method == http.MethodGet && len(parts) == 4 && parts[0] == "api" && parts[1] == "projects" && parts[3] == "dataset-versions":
		r.listDatasetVersions(w, req, parts[2])
	case req.Method == http.MethodPost && len(parts) == 4 && parts[0] == "api" && parts[1] == "projects" && parts[3] == "dataset-versions":
		r.createDatasetVersion(w, req, parts[2])
	case req.Method == http.MethodGet && len(parts) == 4 && parts[0] == "api" && parts[1] == "dataset-versions" && parts[3] == "files":
		r.listDatasetVersionFiles(w, req, parts[2])
	case req.Method == http.MethodGet && len(parts) == 4 && parts[0] == "api" && parts[1] == "dataset-version-files" && parts[3] == "download":
		r.downloadDatasetVersionFile(w, req, parts[2])
	case req.Method == http.MethodGet && len(parts) == 4 && parts[0] == "api" && parts[1] == "jobs" && parts[3] == "events":
		r.listJobEvents(w, req, parts[2])
	case req.Method == http.MethodPost && match(parts, "api", "jobs", "claim"):
		r.claimJob(w, req)
	case req.Method == http.MethodPatch && len(parts) == 4 && parts[0] == "api" && parts[1] == "jobs":
		r.updateJob(w, req, parts[2], parts[3])
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (r *Router) listProjects(w http.ResponseWriter, req *http.Request) {
	projects, err := r.app.ListProjects(req.Context())
	writeResult(w, projects, err)
}

func (r *Router) createProject(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Domain      string `json:"domain"`
	}
	if err := readJSON(req, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	project, err := r.app.CreateProject(req.Context(), body.Name, body.Description, body.Domain)
	writeResult(w, project, err)
}

func (r *Router) getProject(w http.ResponseWriter, req *http.Request, id string) {
	project, err := r.app.GetProject(req.Context(), id)
	writeResult(w, project, err)
}

func (r *Router) listSources(w http.ResponseWriter, req *http.Request, projectID string) {
	sources, err := r.app.ListSources(req.Context(), projectID)
	writeResult(w, sources, err)
}

func (r *Router) uploadSource(w http.ResponseWriter, req *http.Request, projectID string) {
	if err := req.ParseMultipartForm(100 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart upload")
		return
	}
	file, header, err := req.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	source, err := r.app.UploadSource(req.Context(), projectID, header.Filename, file)
	writeResult(w, source, err)
}

func (r *Router) listRuns(w http.ResponseWriter, req *http.Request, projectID string) {
	runs, err := r.app.ListRuns(req.Context(), projectID)
	writeResult(w, runs, err)
}

func (r *Router) startRun(w http.ResponseWriter, req *http.Request, sourceID string) {
	run, err := r.app.StartRun(req.Context(), sourceID)
	writeResult(w, run, err)
}

func (r *Router) listSamples(w http.ResponseWriter, req *http.Request, projectID string) {
	limit := parseInt(req.URL.Query().Get("limit"), 50)
	offset := parseInt(req.URL.Query().Get("offset"), 0)
	status := req.URL.Query().Get("status")
	samples, err := r.app.ListSamples(req.Context(), projectID, status, limit, offset)
	writeResult(w, samples, err)
}

func (r *Router) reviewSample(w http.ResponseWriter, req *http.Request, sampleID string) {
	var body struct {
		Status   string `json:"status"`
		Reviewer string `json:"reviewer"`
		Note     string `json:"note"`
	}
	if err := readJSON(req, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sample, err := r.app.ReviewSample(req.Context(), sampleID, body.Status, body.Reviewer, body.Note)
	writeResult(w, sample, err)
}

func (r *Router) listQualityIssues(w http.ResponseWriter, req *http.Request, projectID string) {
	limit := parseInt(req.URL.Query().Get("limit"), 50)
	issues, err := r.app.ListQualityIssues(req.Context(), projectID, limit)
	writeResult(w, issues, err)
}

func (r *Router) listDatasetVersions(w http.ResponseWriter, req *http.Request, projectID string) {
	versions, err := r.app.ListDatasetVersions(req.Context(), projectID)
	writeResult(w, versions, err)
}

func (r *Router) createDatasetVersion(w http.ResponseWriter, req *http.Request, projectID string) {
	var body struct {
		RunID       string `json:"run_id"`
		VersionName string `json:"version_name"`
	}
	if err := readJSON(req, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	version, err := r.app.CreateDatasetVersion(req.Context(), projectID, body.RunID, body.VersionName)
	writeResult(w, version, err)
}

func (r *Router) claimJob(w http.ResponseWriter, req *http.Request) {
	var body struct {
		WorkerID string `json:"worker_id"`
	}
	_ = readJSON(req, &body)
	job, err := r.app.ClaimJob(req.Context(), body.WorkerID)
	if errors.Is(err, repository.ErrNoJob) {
		writeJSON(w, http.StatusNoContent, nil)
		return
	}
	writeResult(w, job, err)
}

func (r *Router) updateJob(w http.ResponseWriter, req *http.Request, jobID, action string) {
	var body struct {
		Result       json.RawMessage `json:"result"`
		ErrorMessage string          `json:"error_message"`
		Stage        string          `json:"stage"`
		Progress     int             `json:"progress"`
		Message      string          `json:"message"`
		Metadata     json.RawMessage `json:"metadata"`
	}
	_ = readJSON(req, &body)

	var err error
	switch action {
	case "running":
		err = r.app.MarkJobRunning(req.Context(), jobID)
	case "heartbeat":
		err = r.app.HeartbeatJob(req.Context(), jobID)
	case "progress":
		metadata := "{}"
		if len(body.Metadata) > 0 {
			metadata = string(body.Metadata)
		}
		err = r.app.UpdateJobProgress(req.Context(), jobID, body.Stage, body.Progress, body.Message, metadata)
	case "complete":
		result := "{}"
		if len(body.Result) > 0 {
			result = string(body.Result)
		}
		err = r.app.CompleteJob(req.Context(), jobID, result)
	case "fail":
		err = r.app.FailJob(req.Context(), jobID, body.ErrorMessage)
	default:
		writeError(w, http.StatusNotFound, "unknown job action")
		return
	}
	writeResult(w, map[string]string{"status": "ok"}, err)
}

func writeResult(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func readJSON(req *http.Request, value any) error {
	if req.Body == nil {
		return nil
	}
	defer req.Body.Close()
	return json.NewDecoder(req.Body).Decode(value)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func match(parts []string, expected ...string) bool {
	if len(parts) != len(expected) {
		return false
	}
	for i := range expected {
		if parts[i] != expected[i] {
			return false
		}
	}
	return true
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, req)
	})
}
