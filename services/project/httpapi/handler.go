// Package httpapi exposes TASK-016 under the /v1/projects prefix.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"miangedan/services/identity"
	"miangedan/services/project"
	"miangedan/services/region"
)

const maxRequestBodyBytes int64 = 256 << 10

// Application 为传输无关的项目应用接口（对应 openapi projects/plans 标签）。
type Application interface {
	CreateProject(context.Context, project.Actor, project.CreateInput, string) (project.Project, error)
	GetProject(context.Context, project.Actor, string) (project.Project, error)
	ListProjects(context.Context, project.Actor, project.ListFilter) ([]project.Project, error)
	RenameProject(context.Context, project.Actor, string, string, string) (project.Project, error)
	DeleteProject(context.Context, project.Actor, string, string) (project.DeletionTask, error)
	DuplicateProject(context.Context, project.Actor, string, string, string) (project.Project, error)
	GetPlan(context.Context, project.Actor, string) (project.PlanVersion, error)
	GeneratePlanDraft(context.Context, project.Actor, string, string) (project.PlanVersion, error)
	EditPlan(context.Context, project.Actor, string, int, []project.RoundConfig, string) (project.PlanVersion, error)
	ConfirmPlan(context.Context, project.Actor, string, int, []string, string, string) (project.Project, error)
	SaveLibraryEntry(context.Context, project.Actor, project.LibraryKind, string, int, string, string, string) (project.LibraryEntry, error)
	ListLibrary(context.Context, project.Actor, project.LibraryKind) ([]project.LibraryEntry, error)
	DeleteLibraryEntry(context.Context, project.Actor, project.LibraryKind, string, string) error
	GetPreferences(context.Context, project.Actor) (project.Preferences, error)
	SetPreferences(context.Context, project.Actor, string, string, string) (project.Preferences, error)
	ClaimDevice(context.Context, project.Actor, string, string, string) (project.Project, error)
	TransferDevice(context.Context, project.Actor, string, string, string, string) (project.Project, error)
	ReleaseDevice(context.Context, project.Actor, string, string, string) (project.Project, error)
}

// Authenticator 由 TASK-010 identity 服务实现（业务令牌）。
type Authenticator interface {
	Authenticate(string) (identity.Claims, error)
}

// New 构建区域绑定的项目 HTTP 处理器。
func New(app Application, authenticator Authenticator, dataRegion string) (http.Handler, error) {
	if app == nil || authenticator == nil || region.ValidateDataRegion(dataRegion) != nil {
		return nil, errors.New("project http handler requires application, authenticator and valid data region")
	}
	h := &handler{app: app, authenticator: authenticator, dataRegion: dataRegion}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/projects", h.createProject)
	mux.HandleFunc("GET /v1/projects", h.listProjects)
	mux.HandleFunc("GET /v1/projects/{projectId}", h.getProject)
	mux.HandleFunc("PATCH /v1/projects/{projectId}", h.renameProject)
	mux.HandleFunc("DELETE /v1/projects/{projectId}", h.deleteProject)
	mux.HandleFunc("POST /v1/projects/{projectId}/duplicate", h.duplicateProject)
	mux.HandleFunc("POST /v1/projects/{projectId}/plan:generate", h.generatePlan)
	mux.HandleFunc("GET /v1/projects/{projectId}/plan", h.getPlan)
	mux.HandleFunc("PATCH /v1/projects/{projectId}/plan", h.editPlan)
	mux.HandleFunc("POST /v1/projects/{projectId}/plan:confirm", h.confirmPlan)
	mux.HandleFunc("GET /v1/library/resumes", h.listResumes)
	mux.HandleFunc("POST /v1/library/resumes", h.saveResume)
	mux.HandleFunc("DELETE /v1/library/resumes/{resumeId}", h.deleteResume)
	mux.HandleFunc("GET /v1/library/jobs", h.listJobs)
	mux.HandleFunc("POST /v1/library/jobs", h.saveJob)
	mux.HandleFunc("DELETE /v1/library/jobs/{jobId}", h.deleteJob)
	mux.HandleFunc("GET /v1/me/preferences", h.getPreferences)
	mux.HandleFunc("PUT /v1/me/preferences", h.setPreferences)
	mux.HandleFunc("POST /v1/projects/{projectId}/device:claim", h.claimDevice)
	mux.HandleFunc("POST /v1/projects/{projectId}/device:transfer", h.transferDevice)
	mux.HandleFunc("POST /v1/projects/{projectId}/device:release", h.releaseDevice)
	return mux, nil
}

type handler struct {
	app           Application
	authenticator Authenticator
	dataRegion    string
}

func (h *handler) actor(r *http.Request) (project.Actor, error) {
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if auth == "" || token == auth {
		return project.Actor{}, errors.New("unauthorized")
	}
	claims, err := h.authenticator.Authenticate(token)
	if err != nil {
		return project.Actor{}, err
	}
	if claims.DataRegion != h.dataRegion {
		return project.Actor{}, errRegionMismatch
	}
	return project.Actor{UserID: claims.UserID, DataRegion: claims.DataRegion}, nil
}

var errRegionMismatch = errors.New("region_mismatch")

func (h *handler) decode(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must be a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}

func mapError(err error) (int, string, string) {
	switch {
	case errors.Is(err, project.ErrInvalidInput):
		return http.StatusUnprocessableEntity, "invalid_input", err.Error()
	case errors.Is(err, project.ErrStateConflict):
		return http.StatusConflict, "state_conflict", err.Error()
	case errors.Is(err, project.ErrPlanIncomplete):
		return http.StatusUnprocessableEntity, "plan_incomplete", err.Error()
	case errors.Is(err, project.ErrDeviceActive):
		return http.StatusConflict, "device_active", err.Error()
	case errors.Is(err, project.ErrNotFound):
		return http.StatusNotFound, "not_found", err.Error()
	case errors.Is(err, errRegionMismatch):
		return http.StatusForbidden, "region_mismatch", "跨区请求被拒（ADR-0005）"
	default:
		return http.StatusUnauthorized, "unauthorized", err.Error()
	}
}

// ---- 请求/响应结构（对齐 openapi schemas） ----

type projectJSON struct {
	ProjectID            string    `json:"project_id"`
	DataRegion           string    `json:"data_region"`
	Name                 *string   `json:"name"`
	InterviewLanguage    string    `json:"interview_language"`
	DegradedMode         string    `json:"degraded_mode"`
	Status               string    `json:"status"`
	CurrentRoundSequence int       `json:"current_round_sequence"`
	PlanVersion          *int      `json:"plan_version"`
	ActiveDeviceID       *string   `json:"active_device_id"`
	AssignmentID         *string   `json:"assignment_id"`
	CreatedAt            time.Time `json:"created_at"`
}

func toProjectJSON(p project.Project) projectJSON {
	out := projectJSON{
		ProjectID:            p.ProjectID,
		DataRegion:           p.DataRegion,
		InterviewLanguage:    p.InterviewLanguage,
		DegradedMode:         string(p.DegradedMode),
		Status:               string(p.Status),
		CurrentRoundSequence: p.CurrentRoundSequence,
		CreatedAt:            p.CreatedAt,
	}
	if p.Name != "" {
		out.Name = &p.Name
	}
	if p.PlanVersion > 0 {
		v := p.PlanVersion
		out.PlanVersion = &v
	}
	if p.ActiveDeviceID != "" {
		out.ActiveDeviceID = &p.ActiveDeviceID
	}
	if p.AssignmentID != "" {
		out.AssignmentID = &p.AssignmentID
	}
	return out
}

type roundJSON struct {
	Sequence                  int            `json:"sequence"`
	RoundType                 string         `json:"round_type"`
	Role                      *string        `json:"role"`
	Focus                     *string        `json:"focus"`
	DurationMinutes           int            `json:"duration_minutes"`
	Difficulty                string         `json:"difficulty"`
	CriticalDimensions        []string       `json:"critical_dimensions"`
	Tools                     []string       `json:"tools"`
	StyleParameters           map[string]any `json:"style_parameters"`
	AvatarCharacterID         *string        `json:"avatar_character_id"`
	VoiceID                   *string        `json:"voice_id"`
	RubricBound               bool           `json:"rubric_bound"`
	QuestionCoveragePlanReady bool           `json:"question_coverage_plan_ready"`
}

func toRoundJSON(r project.RoundConfig) roundJSON {
	out := roundJSON{
		Sequence:                  r.Sequence,
		RoundType:                 r.RoundType,
		DurationMinutes:           r.DurationMinutes,
		Difficulty:                r.Difficulty,
		CriticalDimensions:        r.CriticalDimensions,
		Tools:                     r.Tools,
		StyleParameters:           r.StyleParameters,
		RubricBound:               r.RubricBound,
		QuestionCoveragePlanReady: r.QuestionCoveragePlanReady,
	}
	if r.Role != "" {
		out.Role = &r.Role
	}
	if r.Focus != "" {
		out.Focus = &r.Focus
	}
	if r.AvatarCharacterID != "" {
		out.AvatarCharacterID = &r.AvatarCharacterID
	}
	if r.VoiceID != "" {
		out.VoiceID = &r.VoiceID
	}
	return out
}

type planJSON struct {
	ProjectID               string              `json:"project_id"`
	PlanVersion             int                 `json:"plan_version"`
	RubricVersion           string              `json:"rubric_version"`
	DimensionWeights        map[string]int      `json:"dimension_weights"`
	Rounds                  []roundJSON         `json:"rounds"`
	RoundWeights            []roundWeightJSON   `json:"round_weights"`
	ProcessSourceRefs       []processSourceJSON `json:"process_source_refs"`
	FlowUsesGenericTemplate bool                `json:"flow_uses_generic_template"`
	Frozen                  bool                `json:"frozen"`
	DataRegion              string              `json:"data_region"`
}

type roundWeightJSON struct {
	Sequence int `json:"sequence"`
	Weight   int `json:"weight"`
}

type processSourceJSON struct {
	SourceID               string    `json:"source_id"`
	SourceType             string    `json:"source_type"`
	URL                    *string   `json:"url"`
	RetrievedAt            time.Time `json:"retrieved_at"`
	Credibility            string    `json:"credibility"`
	IsUnofficialExperience bool      `json:"is_unofficial_experience"`
}

func toPlanJSON(p project.PlanVersion) planJSON {
	out := planJSON{
		ProjectID:               p.ProjectID,
		PlanVersion:             p.PlanVersion,
		RubricVersion:           p.RubricVersion,
		DimensionWeights:        p.DimensionWeights,
		RoundWeights:            []roundWeightJSON{},
		ProcessSourceRefs:       []processSourceJSON{},
		FlowUsesGenericTemplate: p.FlowUsesGenericTemplate,
		Frozen:                  p.Frozen,
		DataRegion:              p.DataRegion,
	}
	for _, r := range p.Rounds {
		out.Rounds = append(out.Rounds, toRoundJSON(r))
	}
	for _, rw := range p.RoundWeights {
		out.RoundWeights = append(out.RoundWeights, roundWeightJSON{Sequence: rw.Sequence, Weight: rw.Weight})
	}
	for _, ps := range p.ProcessSourceRefs {
		item := processSourceJSON{
			SourceID:               ps.SourceID,
			SourceType:             ps.SourceType,
			RetrievedAt:            ps.RetrievedAt,
			Credibility:            ps.Credibility,
			IsUnofficialExperience: ps.IsUnofficialExperience,
		}
		if ps.URL != "" {
			item.URL = &ps.URL
		}
		out.ProcessSourceRefs = append(out.ProcessSourceRefs, item)
	}
	return out
}

// ---- handlers ----

type createRequest struct {
	ResumeID              *string `json:"resume_id"`
	ResumeVersion         *int    `json:"resume_version"`
	JobID                 *string `json:"job_id"`
	JobVersion            *int    `json:"job_version"`
	InterviewLanguage     string  `json:"interview_language"`
	DegradedMode          string  `json:"degraded_mode"`
	DegradedModeConsentID *string `json:"degraded_mode_consent_id"`
	AssignmentID          *string `json:"assignment_id"`
}

func (h *handler) createProject(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	var req createRequest
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	in := project.CreateInput{
		InterviewLanguage: req.InterviewLanguage,
		DegradedMode:      project.DegradedMode(req.DegradedMode),
	}
	if req.ResumeID != nil && req.ResumeVersion != nil {
		in.ResumeRef = &project.MaterialRef{ID: *req.ResumeID, Version: *req.ResumeVersion}
	}
	if req.JobID != nil && req.JobVersion != nil {
		in.JobRef = &project.MaterialRef{ID: *req.JobID, Version: *req.JobVersion}
	}
	if req.DegradedModeConsentID != nil {
		in.DegradedModeConsentID = *req.DegradedModeConsentID
	}
	if req.AssignmentID != nil {
		in.AssignmentID = *req.AssignmentID
	}
	proj, err := h.app.CreateProject(r.Context(), actor, in, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusCreated, toProjectJSON(proj))
}

func (h *handler) getProject(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	proj, err := h.app.GetProject(r.Context(), actor, r.PathValue("projectId"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, toProjectJSON(proj))
}

func (h *handler) listProjects(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	f := project.ListFilter{
		Status:            project.Status(r.URL.Query().Get("status")),
		InterviewLanguage: r.URL.Query().Get("interview_language"),
	}
	if v := r.URL.Query().Get("date_from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_input", "date_from 必须为 RFC3339")
			return
		}
		f.DateFrom = t
	}
	if v := r.URL.Query().Get("date_to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_input", "date_to 必须为 RFC3339")
			return
		}
		f.DateTo = t
	}
	items, err := h.app.ListProjects(r.Context(), actor, f)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	out := make([]projectJSON, 0, len(items))
	for _, p := range items {
		out = append(out, toProjectJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data_region": h.dataRegion, "items": out, "next_cursor": nil})
}

type renameRequest struct {
	Name string `json:"name"`
}

func (h *handler) renameProject(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	var req renameRequest
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	proj, err := h.app.RenameProject(r.Context(), actor, r.PathValue("projectId"), req.Name, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, toProjectJSON(proj))
}

func (h *handler) deleteProject(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	task, err := h.app.DeleteProject(r.Context(), actor, r.PathValue("projectId"), r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"task_id": task.TaskID, "status": task.Status})
}

type duplicateRequest struct {
	InterviewLanguage *string `json:"interview_language"`
}

func (h *handler) duplicateProject(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	var req duplicateRequest
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	lang := ""
	if req.InterviewLanguage != nil {
		lang = *req.InterviewLanguage
	}
	proj, err := h.app.DuplicateProject(r.Context(), actor, r.PathValue("projectId"), lang, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusCreated, toProjectJSON(proj))
}

func (h *handler) generatePlan(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	plan, err := h.app.GeneratePlanDraft(r.Context(), actor, r.PathValue("projectId"), r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusCreated, toPlanJSON(plan))
}

func (h *handler) getPlan(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	plan, err := h.app.GetPlan(r.Context(), actor, r.PathValue("projectId"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, toPlanJSON(plan))
}

type roundInput struct {
	Sequence           int            `json:"sequence"`
	RoundType          string         `json:"round_type"`
	Role               *string        `json:"role"`
	Focus              *string        `json:"focus"`
	DurationMinutes    int            `json:"duration_minutes"`
	Difficulty         string         `json:"difficulty"`
	CriticalDimensions []string       `json:"critical_dimensions"`
	Tools              []string       `json:"tools"`
	StyleParameters    map[string]any `json:"style_parameters"`
	AvatarCharacterID  *string        `json:"avatar_character_id"`
	VoiceID            *string        `json:"voice_id"`
}

func toRoundConfig(in roundInput) project.RoundConfig {
	out := project.RoundConfig{
		Sequence:           in.Sequence,
		RoundType:          in.RoundType,
		DurationMinutes:    in.DurationMinutes,
		Difficulty:         in.Difficulty,
		CriticalDimensions: in.CriticalDimensions,
		Tools:              in.Tools,
		StyleParameters:    in.StyleParameters,
	}
	if in.Role != nil {
		out.Role = *in.Role
	}
	if in.Focus != nil {
		out.Focus = *in.Focus
	}
	if in.AvatarCharacterID != nil {
		out.AvatarCharacterID = *in.AvatarCharacterID
	}
	if in.VoiceID != nil {
		out.VoiceID = *in.VoiceID
	}
	return out
}

type editPlanRequest struct {
	BasePlanVersion int          `json:"base_plan_version"`
	Rounds          []roundInput `json:"rounds"`
}

func (h *handler) editPlan(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	var req editPlanRequest
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	rounds := make([]project.RoundConfig, 0, len(req.Rounds))
	for _, ri := range req.Rounds {
		rounds = append(rounds, toRoundConfig(ri))
	}
	plan, err := h.app.EditPlan(r.Context(), actor, r.PathValue("projectId"), req.BasePlanVersion, rounds, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, toPlanJSON(plan))
}

type confirmRequest struct {
	PlanVersion    int      `json:"plan_version"`
	Accommodations []string `json:"accommodations"`
	QuoteID        *string  `json:"quote_id"`
}

func (h *handler) confirmPlan(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	var req confirmRequest
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	quoteID := ""
	if req.QuoteID != nil {
		quoteID = *req.QuoteID
	}
	proj, err := h.app.ConfirmPlan(r.Context(), actor, r.PathValue("projectId"), req.PlanVersion, req.Accommodations, quoteID, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, toProjectJSON(proj))
}

// ---- 材料库（FR-029） ----

type libraryEntryJSON struct {
	MaterialID string    `json:"material_id"`
	Version    int       `json:"version"`
	Company    *string   `json:"company"`
	JobTitle   *string   `json:"job_title"`
	SavedAt    time.Time `json:"saved_at"`
}

func toLibraryJSON(e project.LibraryEntry) libraryEntryJSON {
	out := libraryEntryJSON{MaterialID: e.MaterialID, Version: e.Version, SavedAt: e.CreatedAt}
	if e.Company != "" {
		out.Company = &e.Company
	}
	if e.JobTitle != "" {
		out.JobTitle = &e.JobTitle
	}
	return out
}

func (h *handler) saveLibrary(w http.ResponseWriter, r *http.Request, kind project.LibraryKind) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	var req struct {
		MaterialID string  `json:"material_id"`
		Version    int     `json:"version"`
		Company    *string `json:"company"`
		JobTitle   *string `json:"job_title"`
	}
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	company, jobTitle := "", ""
	if req.Company != nil {
		company = *req.Company
	}
	if req.JobTitle != nil {
		jobTitle = *req.JobTitle
	}
	entry, err := h.app.SaveLibraryEntry(r.Context(), actor, kind, req.MaterialID, req.Version, company, jobTitle, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusCreated, toLibraryJSON(entry))
}

func (h *handler) listLibrary(w http.ResponseWriter, r *http.Request, kind project.LibraryKind) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	items, err := h.app.ListLibrary(r.Context(), actor, kind)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	out := make([]libraryEntryJSON, 0, len(items))
	for _, e := range items {
		out = append(out, toLibraryJSON(e))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data_region": h.dataRegion, "items": out, "next_cursor": nil})
}

func (h *handler) deleteLibrary(w http.ResponseWriter, r *http.Request, kind project.LibraryKind, id string) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	if err := h.app.DeleteLibraryEntry(r.Context(), actor, kind, id, r.Header.Get("Idempotency-Key")); err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) listResumes(w http.ResponseWriter, r *http.Request) {
	h.listLibrary(w, r, project.KindResume)
}
func (h *handler) saveResume(w http.ResponseWriter, r *http.Request) {
	h.saveLibrary(w, r, project.KindResume)
}
func (h *handler) deleteResume(w http.ResponseWriter, r *http.Request) {
	h.deleteLibrary(w, r, project.KindResume, r.PathValue("resumeId"))
}
func (h *handler) listJobs(w http.ResponseWriter, r *http.Request) {
	h.listLibrary(w, r, project.KindJob)
}
func (h *handler) saveJob(w http.ResponseWriter, r *http.Request) {
	h.saveLibrary(w, r, project.KindJob)
}
func (h *handler) deleteJob(w http.ResponseWriter, r *http.Request) {
	h.deleteLibrary(w, r, project.KindJob, r.PathValue("jobId"))
}

// ---- 语言偏好（FR-028） ----

type preferencesJSON struct {
	UILanguage        string `json:"ui_language"`
	InterviewLanguage string `json:"interview_language"`
}

func (h *handler) getPreferences(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	p, err := h.app.GetPreferences(r.Context(), actor)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, preferencesJSON{UILanguage: p.UILanguage, InterviewLanguage: p.InterviewLanguage})
}

func (h *handler) setPreferences(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	var req preferencesJSON
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	p, err := h.app.SetPreferences(r.Context(), actor, req.UILanguage, req.InterviewLanguage, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, preferencesJSON{UILanguage: p.UILanguage, InterviewLanguage: p.InterviewLanguage})
}

// ---- 单活动设备（FR-030） ----

func (h *handler) claimDevice(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	var req struct {
		DeviceID string `json:"device_id"`
	}
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	proj, err := h.app.ClaimDevice(r.Context(), actor, r.PathValue("projectId"), req.DeviceID, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project_id": proj.ProjectID, "active_device_id": proj.ActiveDeviceID, "claimed": true})
}

func (h *handler) transferDevice(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	var req struct {
		CurrentDeviceID string `json:"current_device_id"`
		NewDeviceID     string `json:"new_device_id"`
	}
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	proj, err := h.app.TransferDevice(r.Context(), actor, r.PathValue("projectId"), req.CurrentDeviceID, req.NewDeviceID, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project_id": proj.ProjectID, "active_device_id": proj.ActiveDeviceID,
		"previous_device_invalidated": true,
	})
}

func (h *handler) releaseDevice(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	var req struct {
		DeviceID string `json:"device_id"`
	}
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	proj, err := h.app.ReleaseDevice(r.Context(), actor, r.PathValue("projectId"), req.DeviceID, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project_id": proj.ProjectID, "active_device_id": proj.ActiveDeviceID, "released": true})
}
