package index

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// ProjectGraphNodes reads all current (valid_to IS NULL) nodes from the graph
// database and upserts them into fts_nodes + node_index. Returns the number of
// nodes projected.
func (idx *DB) ProjectGraphNodes(graphDB *sql.DB) (int, error) {
	if graphDB == nil {
		return 0, nil
	}

	rows, err := graphDB.Query(`
		SELECT type, key, props, valid_from, COALESCE(
			(SELECT GROUP_CONCAT(e.type || ':' || n2.key, ',')
			 FROM edges e JOIN nodes n2 ON n2.id = e.to_id
			 WHERE e.from_id = nodes.id AND e.valid_to IS NULL),
		'') as tags
		FROM nodes
		WHERE valid_to IS NULL
	`)
	if err != nil {
		return 0, fmt.Errorf("querying graph nodes: %w", err)
	}
	defer rows.Close()

	tx, err := idx.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Clear existing projected data.
	for _, table := range []string{"fts_nodes", "node_index"} {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return 0, fmt.Errorf("clearing %s: %w", table, err)
		}
	}

	insertFTS, err := tx.Prepare("INSERT INTO fts_nodes(rowid, title, body) VALUES (?, ?, ?)")
	if err != nil {
		return 0, fmt.Errorf("prepare fts insert: %w", err)
	}
	defer insertFTS.Close()

	insertMeta, err := tx.Prepare(`INSERT INTO node_index(rowid, node_type, key, tags, valid_from)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare meta insert: %w", err)
	}
	defer insertMeta.Close()

	var count int
	var rowID int64 = 1

	for rows.Next() {
		var nodeType, key, propsJSON, validFrom, tags string
		if err := rows.Scan(&nodeType, &key, &propsJSON, &validFrom, &tags); err != nil {
			return count, fmt.Errorf("scanning node: %w", err)
		}

		var props map[string]any
		if err := json.Unmarshal([]byte(propsJSON), &props); err != nil {
			props = map[string]any{}
		}

		title := propString(props, "title")
		if title == "" {
			title = propString(props, "subject")
		}
		if title == "" {
			title = key
		}

		body := propString(props, "body")
		if body == "" {
			body = propString(props, "description")
		}
		if body == "" {
			body = propString(props, "content")
		}

		if _, err := insertFTS.Exec(rowID, title, body); err != nil {
			return count, fmt.Errorf("insert fts node %s/%s: %w", nodeType, key, err)
		}
		if _, err := insertMeta.Exec(rowID, nodeType, key, tags, validFrom); err != nil {
			return count, fmt.Errorf("insert node_index %s/%s: %w", nodeType, key, err)
		}

		rowID++
		count++
	}

	if err := rows.Err(); err != nil {
		return count, fmt.Errorf("iterating nodes: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return count, fmt.Errorf("commit projection: %w", err)
	}

	return count, nil
}

func propString(props map[string]any, key string) string {
	v, ok := props[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
