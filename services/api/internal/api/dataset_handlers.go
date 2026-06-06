package api

import (
	"net/http"
	"os"
)

func (r *Router) listProjectJobs(w http.ResponseWriter, req *http.Request, projectID string) {
	jobs, err := r.app.ListProjectJobs(req.Context(), projectID)
	writeResult(w, jobs, err)
}

func (r *Router) listJobEvents(w http.ResponseWriter, req *http.Request, jobID string) {
	events, err := r.app.ListJobEvents(req.Context(), jobID)
	writeResult(w, events, err)
}

func (r *Router) editSample(w http.ResponseWriter, req *http.Request, sampleID string) {
	var body struct {
		InputText    string `json:"input_text"`
		OutputText   string `json:"output_text"`
		EditedBy     string `json:"edited_by"`
		ChangeReason string `json:"change_reason"`
		Status       string `json:"status"`
	}
	if err := readJSON(req, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sample, err := r.app.EditSample(req.Context(), sampleID, body.InputText, body.OutputText, body.EditedBy, body.ChangeReason, body.Status)
	writeResult(w, sample, err)
}

func (r *Router) listSampleVersions(w http.ResponseWriter, req *http.Request, sampleID string) {
	versions, err := r.app.ListSampleVersions(req.Context(), sampleID)
	writeResult(w, versions, err)
}

func (r *Router) listDatasetVersionFiles(w http.ResponseWriter, req *http.Request, versionID string) {
	files, err := r.app.ListDatasetVersionFiles(req.Context(), versionID)
	writeResult(w, files, err)
}

func (r *Router) downloadDatasetVersionFile(w http.ResponseWriter, req *http.Request, fileID string) {
	file, err := r.app.GetDatasetVersionFile(req.Context(), fileID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if _, err := os.Stat(file.FilePath); err != nil {
		writeError(w, http.StatusNotFound, "dataset version file not found")
		return
	}
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+file.FileName+`"`)
	http.ServeFile(w, req, file.FilePath)
}
