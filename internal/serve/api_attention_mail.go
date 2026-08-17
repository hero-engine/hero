package serve

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/contracts/attention/mailread"
)

func (a *API) handleAttentionMailMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	request := mailread.ListRequest{
		SchemaVersion: mailread.SchemaVersion,
		ProjectPeerID: r.URL.Query().Get("project_peer_id"),
		ThreadID:      r.URL.Query().Get("thread_id"),
		Cursor:        r.URL.Query().Get("cursor"),
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			writeMailListResponse(w, mailread.ListResponse{SchemaVersion: mailread.SchemaVersion, Error: mailValidation("limit must be an integer", "limit")})
			return
		}
		request.Limit = limit
	}
	if raw, present := r.URL.Query()["unread_only"]; present {
		if len(raw) != 1 {
			writeMailListResponse(w, mailread.ListResponse{SchemaVersion: mailread.SchemaVersion, Error: mailValidation("unread_only must appear once", "unread_only")})
			return
		}
		value, err := strconv.ParseBool(raw[0])
		if err != nil {
			writeMailListResponse(w, mailread.ListResponse{SchemaVersion: mailread.SchemaVersion, Error: mailValidation("unread_only must be true or false", "unread_only")})
			return
		}
		request.UnreadOnly = &value
	}
	service, err := a.loadMailQueryService()
	if err != nil {
		writeMailListResponse(w, mailread.ListResponse{SchemaVersion: mailread.SchemaVersion, Error: mailUnavailable(err.Error())})
		return
	}
	writeMailListResponse(w, service.List(request))
}

func (a *API) handleAttentionMailMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	messageID := strings.TrimPrefix(r.URL.Path, "/api/attention/v1/mail/messages/")
	if messageID == "" || strings.Contains(messageID, "/") {
		writeMailDetailResponse(w, mailread.DetailResponse{SchemaVersion: mailread.SchemaVersion, Error: mailValidation("message_id is required", "message_id")})
		return
	}
	service, err := a.loadMailQueryService()
	if err != nil {
		writeMailDetailResponse(w, mailread.DetailResponse{SchemaVersion: mailread.SchemaVersion, Error: mailUnavailable(err.Error())})
		return
	}
	writeMailDetailResponse(w, service.Detail(r.URL.Query().Get("project_peer_id"), messageID))
}

func (a *API) handleAttentionMailAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request mailread.ActionRequest
	if contractErr := decodeMailJSON(w, r, 128<<10, &request); contractErr != nil {
		writeMailActionResponse(w, mailread.ActionResponse{SchemaVersion: mailread.SchemaVersion, Error: contractErr})
		return
	}
	service, err := a.loadMailQueryService()
	if err != nil {
		writeMailActionResponse(w, mailread.ActionResponse{SchemaVersion: mailread.SchemaVersion, Error: mailUnavailable(err.Error())})
		return
	}
	writeMailActionResponse(w, service.Action(request))
}

func (a *API) handleAttentionMailReply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request mailread.ReplyRequest
	if contractErr := decodeMailJSON(w, r, 128<<10, &request); contractErr != nil {
		writeMailReplyResponse(w, mailread.ReplyResponse{SchemaVersion: mailread.SchemaVersion, Error: contractErr})
		return
	}
	service, err := a.loadMailQueryService()
	if err != nil {
		writeMailReplyResponse(w, mailread.ReplyResponse{SchemaVersion: mailread.SchemaVersion, Error: mailUnavailable(err.Error())})
		return
	}
	writeMailReplyResponse(w, service.Reply(request))
}

func (a *API) handleAttentionMailContract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, mailread.ContractResponse{
		SchemaVersion: mailread.SchemaVersion, BundleVersion: mailread.BundleVersion,
		BundleManifestSHA256: mailread.ConformanceManifestSHA256, Compatibility: mailread.Compatibility,
	})
}

func (a *API) loadMailQueryService() (mailQueryService, error) {
	if a.mailQueryService == nil {
		return nil, &attention.ContractError{Code: attention.ErrorUnavailable, Message: "Mail query service is unavailable"}
	}
	return a.mailQueryService()
}

func decodeMailJSON(w http.ResponseWriter, r *http.Request, maxBytes int64, destination any) *attention.ContractError {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	if err := decoder.Decode(destination); err != nil {
		return mailValidation("invalid request JSON", "request")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return mailValidation("request must contain one JSON value", "request")
	}
	return nil
}

func writeMailListResponse(w http.ResponseWriter, response mailread.ListResponse) {
	writeJSON(w, mailResponseStatus(response.Error), response)
}

func writeMailDetailResponse(w http.ResponseWriter, response mailread.DetailResponse) {
	writeJSON(w, mailResponseStatus(response.Error), response)
}

func writeMailActionResponse(w http.ResponseWriter, response mailread.ActionResponse) {
	writeJSON(w, mailResponseStatus(response.Error), response)
}

func writeMailReplyResponse(w http.ResponseWriter, response mailread.ReplyResponse) {
	writeJSON(w, mailResponseStatus(response.Error), response)
}

func mailResponseStatus(contractErr *attention.ContractError) int {
	if contractErr == nil {
		return http.StatusOK
	}
	return attentionStatus(contractErr.Code)
}

func mailValidation(message, field string) *attention.ContractError {
	return &attention.ContractError{Code: attention.ErrorValidation, Message: message, Field: field}
}

func mailUnavailable(message string) *attention.ContractError {
	return &attention.ContractError{Code: attention.ErrorUnavailable, Message: message}
}
