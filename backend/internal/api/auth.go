package api

import (
	"net/http"

	"moneytracker/backend/internal/auth"
	"moneytracker/backend/internal/notion"
)

// meHandler tells the SPA whether login is required and who is signed in.
// Mounted outside the auth middleware: the login screen depends on it.
func meHandler(a *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.Enabled() {
			writeJSON(w, http.StatusOK, map[string]any{"enabled": false, "authenticated": false})
			return
		}
		acc, ok := a.CurrentUser(r)
		if !ok {
			writeJSON(w, http.StatusOK, map[string]any{"enabled": true, "authenticated": false})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled":       true,
			"authenticated": true,
			"user": map[string]string{
				"name":          acc.Name,
				"email":         acc.Email,
				"avatarUrl":     acc.AvatarURL,
				"workspaceName": acc.WorkspaceName,
			},
		})
	}
}

func notionStatusHandler(a *auth.Service, y *notion.Syncer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.Enabled() {
			writeJSON(w, http.StatusOK, map[string]any{"configured": false})
			return
		}
		st := y.Status(auth.UserID(r.Context()))
		writeJSON(w, http.StatusOK, map[string]any{
			"configured":    true,
			"connected":     st.Connected,
			"workspaceName": st.WorkspaceName,
			"pageUrl":       st.PageURL,
			"running":       st.Running,
			"lastSyncedAt":  st.LastSyncedAt,
			"last":          st.Last,
		})
	}
}

func notionSyncHandler(a *auth.Service, y *notion.Syncer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.Enabled() {
			writeError(w, http.StatusBadRequest, "Notion login is not configured on this server")
			return
		}
		if err := y.Start(auth.UserID(r.Context())); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "sync started"})
	}
}
