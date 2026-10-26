package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"tourna-desk/internal/i18n"
	"tourna-desk/internal/models"
	"tourna-desk/internal/service"
)

type APIHandler struct {
	svc     *service.TournamentService
	localIP string
	port    string
}

func NewAPIHandler(svc *service.TournamentService, localIP, port string) *APIHandler {
	return &APIHandler{
		svc:     svc,
		localIP: localIP,
		port:    port,
	}
}

func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/info", h.handleGetInfo)
	mux.HandleFunc("GET /api/venues", h.handleListVenues)
	mux.HandleFunc("POST /api/venues", h.handleCreateVenue)
	mux.HandleFunc("PUT /api/venues/{id}", h.handleUpdateVenue)
	mux.HandleFunc("DELETE /api/venues/{id}", h.handleDeleteVenue)
	mux.HandleFunc("GET /api/tournaments", h.handleListTournaments)
	mux.HandleFunc("POST /api/tournaments", h.handleCreateTournament)
	mux.HandleFunc("DELETE /api/tournaments/{id}", h.handleDeleteTournament)

	mux.HandleFunc("POST /api/categories", h.handleCreateCategory)
	mux.HandleFunc("GET /api/categories/{id}", h.handleGetCategoryDetail)
	mux.HandleFunc("PUT /api/categories/{id}", h.handleUpdateCategory)
	mux.HandleFunc("DELETE /api/categories/{id}", h.handleDeleteCategory)
	mux.HandleFunc("POST /api/categories/{id}/stages", h.handleCreateStage)
	mux.HandleFunc("POST /api/categories/{id}/teams", h.handleCreateTeam)
	mux.HandleFunc("POST /api/categories/{id}/teams/import-excel", h.handleImportTeamsExcel)
	mux.HandleFunc("POST /api/categories/{id}/teams/auto-distribute", h.handleAutoDistributeTeams)
	mux.HandleFunc("POST /api/categories/{id}/initial-draw", h.handleInitialDraw)
	mux.HandleFunc("GET /api/categories/{id}/pots", h.handleGetPots)
	mux.HandleFunc("POST /api/categories/{id}/pots/prepare", h.handlePreparePots)
	mux.HandleFunc("PUT /api/categories/{id}/pots", h.handleSavePots)
	mux.HandleFunc("POST /api/categories/{id}/next-round-draw", h.handleNextRoundDraw)
	mux.HandleFunc("POST /api/categories/{id}/final-four", h.handleBuildFinalFour)
	mux.HandleFunc("POST /api/categories/{id}/knockout", h.handleBuildKnockout)
	mux.HandleFunc("POST /api/categories/{id}/matches/swap", h.handleSwapMatchPairings)
	mux.HandleFunc("POST /api/categories/{id}/generate-fixtures", h.handleGenerateFixtures)
	mux.HandleFunc("POST /api/categories/{id}/reset-fixtures", h.handleResetFixtures)
	mux.HandleFunc("GET /api/categories/{id}/export-excel", h.handleExportExcel)
	mux.HandleFunc("GET /api/categories/{id}/export-bracket-excel", h.handleExportBracketExcel)

	mux.HandleFunc("GET /api/teams/template-excel", h.handleDownloadTeamTemplate)
	mux.HandleFunc("PUT /api/teams/{id}/stage", h.handleUpdateTeamStage)
	mux.HandleFunc("PUT /api/teams/{id}/seed", h.handleUpdateTeamSeed)
	mux.HandleFunc("DELETE /api/stages/{id}", h.handleDeleteStage)
	mux.HandleFunc("DELETE /api/teams/{id}", h.handleDeleteTeam)
	mux.HandleFunc("PUT /api/matches/{id}", h.handleUpdateMatch)
}

func (h *APIHandler) handleGetInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"local_ip":    h.localIP,
		"port":        h.port,
		"network_url": fmt.Sprintf("http://%s:%s", h.localIP, h.port),
		"sports":      h.svc.GetSupportedSports(),
	})
}

func (h *APIHandler) handleListVenues(w http.ResponseWriter, r *http.Request) {
	sport := r.URL.Query().Get("sport")
	venues, err := h.svc.ListVenues(sport)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, venues)
}

func (h *APIHandler) handleCreateVenue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name            string   `json:"name"`
		Address         string   `json:"address"`
		SupportedSports []string `json:"supported_sports"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	v, err := h.svc.CreateVenue(req.Name, req.Address, req.SupportedSports)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (h *APIHandler) handleUpdateVenue(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid venue id")
		return
	}
	var req struct {
		Name            string   `json:"name"`
		Address         string   `json:"address"`
		SupportedSports []string `json:"supported_sports"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	v, err := h.svc.UpdateVenue(id, req.Name, req.Address, req.SupportedSports)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *APIHandler) handleDeleteVenue(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid venue id")
		return
	}
	if err := h.svc.DeleteVenue(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *APIHandler) handleListTournaments(w http.ResponseWriter, r *http.Request) {
	tournaments, err := h.svc.ListTournaments()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tournaments)
}

func (h *APIHandler) handleCreateTournament(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Location    string `json:"location"`
		StartDate   string `json:"start_date"`
		EndDate     string `json:"end_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	t, err := h.svc.CreateTournament(req.Name, req.Description, req.Location, req.StartDate, req.EndDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *APIHandler) handleDeleteTournament(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament id")
		return
	}
	if err := h.svc.DeleteTournament(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *APIHandler) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TournamentID     int64            `json:"tournament_id"`
		SportSlug        models.SportSlug `json:"sport_slug"`
		Name             string           `json:"name"`
		Format           string           `json:"format"`
		InitialStageName string           `json:"initial_stage_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	formatArg := req.Format
	if formatArg == "" {
		formatArg = req.InitialStageName
	}
	cat, err := h.svc.CreateCategory(req.TournamentID, req.SportSlug, req.Name, formatArg)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cat)
}

func (h *APIHandler) handleGetCategoryDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	detail, err := h.svc.GetCategoryDetail(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *APIHandler) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	if err := h.svc.DeleteCategory(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *APIHandler) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var req struct {
		SportSlug models.SportSlug `json:"sport_slug"`
		Name      string           `json:"name"`
		Format    string           `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	detail, err := h.svc.UpdateCategory(id, req.SportSlug, req.Name, req.Format)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *APIHandler) handleCreateStage(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var req struct {
		Name             string `json:"name"`
		DoubleRoundRobin bool   `json:"double_round_robin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	stg, err := h.svc.CreateStage(catID, req.Name, req.DoubleRoundRobin)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, stg)
}

func (h *APIHandler) handleDeleteStage(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage id")
		return
	}
	if err := h.svc.DeleteStage(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *APIHandler) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var req struct {
		StageID    *int64 `json:"stage_id"`
		SchoolName string `json:"school_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	team, err := h.svc.CreateTeam(catID, req.StageID, req.SchoolName)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, team)
}

func (h *APIHandler) handleUpdateTeamStage(w http.ResponseWriter, r *http.Request) {
	teamID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid team id")
		return
	}
	var req struct {
		StageID *int64 `json:"stage_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if err := h.svc.UpdateTeamStage(teamID, req.StageID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *APIHandler) handleUpdateTeamSeed(w http.ResponseWriter, r *http.Request) {
	teamID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid team id")
		return
	}
	var req struct {
		IsSeeded bool `json:"is_seeded"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if err := h.svc.UpdateTeamSeed(teamID, req.IsSeeded); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *APIHandler) handleInitialDraw(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var req service.InitialDrawInput
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
	}
	detail, err := h.svc.ExecuteInitialSeededDraw(catID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *APIHandler) handleGetPots(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	detail, err := h.svc.GetCategoryDetail(catID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pot_assignments": detail.PotAssignments,
		"current_round":   detail.CurrentRound,
	})
}

func (h *APIHandler) handlePreparePots(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var req struct {
		StageIDs []int64 `json:"stage_ids"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	detail, err := h.svc.PrepareCategoryPots(catID, req.StageIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *APIHandler) handleSavePots(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var req service.SavePotsInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	detail, err := h.svc.SaveCategoryPots(catID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *APIHandler) handleNextRoundDraw(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var req service.NextRoundDrawInput
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
	}
	detail, err := h.svc.ExecuteCategoryNextRoundDraw(catID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *APIHandler) handleBuildFinalFour(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var req service.FinalFourInput
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
	}
	detail, err := h.svc.BuildCategoryFinalFour(catID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *APIHandler) handleBuildKnockout(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var req service.KnockoutStageInput
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
	}
	detail, err := h.svc.BuildCategoryKnockout(catID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *APIHandler) handleSwapMatchPairings(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var req service.SwapMatchPairingsInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	detail, err := h.svc.SwapMatchPairings(catID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *APIHandler) handleAutoDistributeTeams(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	if err := h.svc.AutoDistributeTeams(catID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	detail, err := h.svc.GetCategoryDetail(catID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *APIHandler) handleImportTeamsExcel(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}

	var fileReader io.Reader
	filename := r.Header.Get("X-File-Name")
	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			writeError(w, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
			return
		}
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "missing 'file' field in multipart form")
			return
		}
		defer file.Close()
		fileReader = file
		if fileHeader != nil && fileHeader.Filename != "" {
			filename = fileHeader.Filename
		}
	} else {
		fileReader = r.Body
	}

	summary, err := h.svc.ImportTeamsWithSummary(catID, filename, fileReader)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	detail, err := h.svc.GetCategoryDetail(catID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"imported_count": summary.ImportedCount,
		"skipped_count":  summary.SkippedCount,
		"teams":          summary.Teams,
		"imported_teams": summary.Teams,
		"detail":         detail,
	})
}

func (h *APIHandler) handleDownloadTeamTemplate(w http.ResponseWriter, r *http.Request) {
	lang := i18n.DetectLanguage(r)
	var buf bytes.Buffer
	if err := h.svc.ExportTeamTemplateExcel(&buf, lang); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate template: "+err.Error())
		return
	}

	filename := "Takim_Sablonu.xlsx"
	if lang == i18n.LangEN {
		filename = "Team_Import_Template.xlsx"
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func (h *APIHandler) handleDeleteTeam(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid team id")
		return
	}
	if err := h.svc.DeleteTeam(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *APIHandler) handleGenerateFixtures(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var req service.GenerateFixtureInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if _, err := h.svc.GenerateStageFixtures(catID, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	detail, err := h.svc.GetCategoryDetail(catID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *APIHandler) handleResetFixtures(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var req struct {
		StageID *int64 `json:"stage_id"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	detail, err := h.svc.ResetCategoryFixtures(catID, req.StageID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *APIHandler) handleUpdateMatch(w http.ResponseWriter, r *http.Request) {
	matchID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid match id")
		return
	}
	var req service.UpdateMatchInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	match, err := h.svc.UpdateMatch(matchID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	detail, err := h.svc.GetCategoryDetail(match.CategoryID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

var nonAlphaNumRegex = regexp.MustCompile(`[^\p{L}\p{N}_-]+`)

func sanitizeFilename(name string) string {
	s := strings.TrimSpace(name)
	s = strings.ReplaceAll(s, " ", "_")
	s = nonAlphaNumRegex.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "Category"
	}
	return s
}

func (h *APIHandler) handleExportExcel(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	lang := i18n.DetectLanguage(r)

	detail, err := h.svc.GetCategoryDetail(catID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var buf bytes.Buffer
	if err := h.svc.ExportCategoryExcel(&buf, detail, lang); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate excel: "+err.Error())
		return
	}

	safeName := sanitizeFilename(detail.Category.Name)
	filename := fmt.Sprintf("%s_%s.xlsx", safeName, string(lang))

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func (h *APIHandler) handleExportBracketExcel(w http.ResponseWriter, r *http.Request) {
	catID, err := parseID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category id")
		return
	}
	lang := i18n.DetectLanguage(r)

	detail, err := h.svc.GetCategoryDetail(catID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var buf bytes.Buffer
	if err := h.svc.ExportKnockoutBracketExcel(&buf, detail, lang); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate bracket excel: "+err.Error())
		return
	}

	safeName := sanitizeFilename(detail.Category.Name)
	filename := fmt.Sprintf("%s_Bracket_%s.xlsx", safeName, string(lang))

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func parseID(r *http.Request, key string) (int64, error) {
	val := r.PathValue(key)
	return strconv.ParseInt(val, 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
