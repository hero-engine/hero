package serve

func (s *MCPServer) toolAttentionContract(_ map[string]interface{}) (string, error) {
	return marshalSuggestion(attentionContractMetadata())
}
