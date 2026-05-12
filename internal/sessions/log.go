package sessions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// ReadEvents reads all events from a session log file.
func ReadEvents(heroDir, sessionID string) ([]map[string]interface{}, error) {
	f, err := os.Open(LogPath(heroDir, sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()

	var events []map[string]interface{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var evt map[string]interface{}
		if err := json.Unmarshal(line, &evt); err != nil {
			continue // skip malformed lines
		}
		events = append(events, evt)
	}
	return events, scanner.Err()
}
