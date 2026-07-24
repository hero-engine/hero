package serve

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/hero-engine/hero/contracts/attention"
)

func directActionVersion(args map[string]interface{}) (int, *attention.ContractError) {
	version, err := int64Arg(args, "schema_version")
	if err != nil {
		return 0, &attention.ContractError{Code: attention.ErrorValidation, Message: err.Error(), Field: "schema_version"}
	}
	return int(version), nil
}

func directActionSuccess(source any) (string, error) {
	raw, err := json.Marshal(source)
	if err != nil {
		return "", err
	}
	return marshalSuggestion(attention.ActionResult{SchemaVersion: attention.SchemaVersion, Source: raw})
}

func directActionFailure(err *attention.ContractError) (string, error) {
	return marshalSuggestion(attention.ActionResult{SchemaVersion: attention.SchemaVersion, Error: err})
}

func directActionUnavailable(err error) (string, error) {
	return directActionFailure(&attention.ContractError{Code: attention.ErrorUnavailable, Message: err.Error()})
}

func directActionProvenance(kind, id string) []attention.ProvenanceReference {
	if kind == "" && id == "" {
		return nil
	}
	return []attention.ProvenanceReference{{Kind: kind, SourceID: id}}
}

func directActionOriginKey(operationID, key string) string {
	sum := sha256.Sum256([]byte(operationID + "\x00" + key))
	return fmt.Sprintf("mcp:%s:%x", operationID, sum)
}
