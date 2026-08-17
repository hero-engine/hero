package serve

import (
	"encoding/json"
	"net/http"

	"github.com/hero-engine/hero/contracts/attention"
)

const attentionFixtureManifestSHA256 = "059b5418cc2005d506b0ca718df1ed25109ed022e385c7b16bca3e3c4a0d8e07"
const attentionBundleVersion = 1
const attentionBundleManifestSHA256 = "61dd0d42af00df76919b1b35a303f24da51afe25d72740799ae6ca5da81a42a8"
const attentionBundlePath = "contracts/attention/conformance/v1"

func (a *API) handleAttentionSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	service, err := a.loadAttentionService()
	if err != nil {
		writeAttentionFailure(w, &attention.ContractError{Code: attention.ErrorUnavailable, Message: err.Error()})
		return
	}
	snapshot, err := service.Snapshot()
	if err != nil {
		if contractErr, ok := err.(*attention.ContractError); ok {
			writeAttentionFailure(w, contractErr)
			return
		}
		writeAttentionFailure(w, &attention.ContractError{Code: attention.ErrorUnavailable, Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (a *API) handleAttentionAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request attention.ActionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
	if err := decoder.Decode(&request); err != nil {
		writeAttentionFailure(w, &attention.ContractError{Code: attention.ErrorValidation, Message: "invalid action request JSON", Field: "request"})
		return
	}
	service, err := a.loadAttentionService()
	if err != nil {
		writeAttentionFailure(w, &attention.ContractError{Code: attention.ErrorUnavailable, Message: err.Error()})
		return
	}
	result := service.Dispatch(request)
	if result.Error != nil {
		writeJSON(w, attentionStatus(result.Error.Code), result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) handleAttentionContract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, attentionContractMetadata())
}

func attentionContractMetadata() map[string]any {
	return map[string]any{
		"schema_version":          attention.SchemaVersion,
		"fixture_manifest_sha256": attentionFixtureManifestSHA256,
		"bundle_version":          attentionBundleVersion,
		"bundle_manifest_sha256":  attentionBundleManifestSHA256,
		"bundle_path":             attentionBundlePath,
		"compatibility":           "Unknown additive fields and identifiers remain inert but decodable and never grant executable behavior.",
		"synchronization":         "snapshot_refresh",
	}
}

func (a *API) loadAttentionService() (interface {
	Snapshot() (attention.AttentionSnapshot, error)
	Dispatch(attention.ActionRequest) attention.ActionResult
}, error) {
	if a.attentionService == nil {
		return nil, &attention.ContractError{Code: attention.ErrorUnavailable, Message: "attention service is unavailable"}
	}
	return a.attentionService()
}

func writeAttentionFailure(w http.ResponseWriter, contractErr *attention.ContractError) {
	writeJSON(w, attentionStatus(contractErr.Code), attention.ActionResult{
		SchemaVersion: attention.SchemaVersion,
		Error:         contractErr,
	})
}

func attentionStatus(code string) int {
	switch code {
	case attention.ErrorValidation:
		return http.StatusBadRequest
	case attention.ErrorStale:
		return http.StatusConflict
	case attention.ErrorUnsupported:
		return http.StatusUnprocessableEntity
	case attention.ErrorMissing:
		return http.StatusNotFound
	case attention.ErrorIncompatibleVersion:
		return http.StatusUpgradeRequired
	case attention.ErrorUnavailable:
		return http.StatusServiceUnavailable
	case attention.ErrorIdempotencyConflict:
		return http.StatusConflict
	case attention.ErrorPermission:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
