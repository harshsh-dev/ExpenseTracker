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
		out := map[string]any{"enabled": true, "mode": a.Mode(), "authenticated": false}
		if _, ok := a.SessionSubject(r); ok {
			out["authenticated"] = true
			if acc, ok := a.CurrentUser(r); ok {
				out["user"] = map[string]string{
					"name":          acc.Name,
					"email":         acc.Email,
					"avatarUrl":     acc.AvatarURL,
					"workspaceName": acc.WorkspaceName,
				}
			}
		}
		writeJSON(w, http.StatusOK, out)
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
			"lastPull":      st.LastPull,
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

func notionPullHandler(a *auth.Service, y *notion.Syncer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.Enabled() {
			writeError(w, http.StatusBadRequest, "Notion login is not configured on this server")
			return
		}
		if err := y.StartPull(auth.UserID(r.Context())); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "pull started"})
	}
}
