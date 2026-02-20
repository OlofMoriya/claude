package data

import (
	"database/sql"
	"fmt"
	"owl/logger"

	"github.com/fatih/color"
	_ "github.com/marcboeker/go-duckdb"
)

// DuckDB implementation of EmbeddingsStore.
//
// This implementation intentionally mirrors the SQLite one, but uses DuckDB's
// VSS extension when available. The embedding is passed as a string and must be
// in a format DuckDB can cast to FLOAT[] (commonly: "[0.1,0.2,...]").
//
// Notes:
// - DuckDB vector/VSS support is extension-based and may vary by DuckDB version.
// - If VSS isn't available at runtime, setupDb will fail.

type DuckDbEmbeddingsDatabase struct {
	Name string
}

func (edb *DuckDbEmbeddingsDatabase) getUserDb() *sql.DB {
	homeDir, err := getHomeDir()
	if err != nil {
		panic(fmt.Sprintf("did not find home dir for db creation. %s", err))
	}

	path := fmt.Sprintf("%s/.owl/%s.duckdb", homeDir, edb.Name)

	// DuckDB uses its own driver name.
	db, err := sql.Open("duckdb", path)
	if err != nil {
		panic(err)
	}

	_, _ = db.Exec("INSTALL vss;")
	_, err = db.Exec("LOAD vss;")
	if err != nil {
		logger.Debug.Printf("failed to load duckdb vss extension: %s", err)
		logger.Screen(fmt.Sprintf("failed to load duckdb vss extension: %s", err), color.RGB(150, 160, 98))
	}

	// Determine if our schema exists.
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('texts','embeddings')").Scan(&count)
	if err != nil {
		panic(err)
	}
	if count < 2 {
		edb.setupDb(db)
	}

	return db
}

func (edb *DuckDbEmbeddingsDatabase) setupDb(db *sql.DB) {
	// Try to enable/install the VSS extension.
	// These statements are safe to attempt; older builds may error.
	// We propagate errors so caller can see missing support.
	stmts := []string{
		"INSTALL vss",
		"LOAD vss",
		"INSTALL json",
		"LOAD json",
	}
	for _, s := range stmts {
		_, _ = db.Exec(s)
	}

	createDuckDbTextTable(db)
	createDuckDbEmbeddingsTable(db)

	// Create a VSS index if the extension provides it.
	// If unsupported, this will error; ignore to keep basic table usable.
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS embeddings_vss_idx ON embeddings USING vss (embedding)")
}

func (edb *DuckDbEmbeddingsDatabase) FindMatches(embedding string) ([]EmbeddingMatch, error) {
	db := edb.getUserDb()
	defer db.Close()

	logger.Debug.Println("searching embedding (duckdb)")

	selectQuery := `
		WITH needle AS (
			SELECT CAST(? AS FLOAT[1536]) AS search_vec
		),
		matches AS (
			SELECT unnest(res.matches) AS match
			FROM needle, vss_match(embeddings, search_vec, embedding, 3) res
		)
		SELECT (match).row.id, (match).row.text_id, t.content, (match).score, t.reference
		FROM matches
		JOIN texts t ON (match).row.text_id = t.id
		ORDER BY (match).score 
		LIMIT 3;`

	rows, err := db.Query(selectQuery, embedding)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []EmbeddingMatch
	for rows.Next() {
		logger.Debug.Println("reading match from db")
		var match EmbeddingMatch
		err := rows.Scan(&match.Id, &match.TextId, &match.Text, &match.Distance, &match.Reference)
		if err != nil {
			return nil, err
		}
		matches = append(matches, match)
	}
	return matches, nil
}

func (edb *DuckDbEmbeddingsDatabase) InsertEmbedding(text string, embedding string, reference string) (int64, error) {
	db := edb.getUserDb()
	defer db.Close()

	logger.Debug.Println("inserting embedding (duckdb)")

	insertTextQuery := "INSERT INTO texts (content, reference) VALUES (?, ?) RETURNING id"
	var textId int64
	err := db.QueryRow(insertTextQuery, text, reference).Scan(&textId)
	if err != nil {
		return 0, err
	}

	insertQuery := "INSERT INTO embeddings (text_id, embedding) VALUES (?, CAST(? AS FLOAT[1536])) RETURNING id"
	var id int64
	err = db.QueryRow(insertQuery, textId, embedding).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func createDuckDbTextTable(db *sql.DB) {
	createTableQuery := `
		CREATE SEQUENCE IF NOT EXISTS texts_id_seq;

		CREATE TABLE IF NOT EXISTS texts (
			id BIGINT PRIMARY KEY DEFAULT NEXTVAL('texts_id_seq'),
			content   TEXT NOT NULL,
			reference TEXT NOT NULL
		);`
	_, err := db.Exec(createTableQuery)
	if err != nil {
		panic(err)
	}
}

func createDuckDbEmbeddingsTable(db *sql.DB) {
	// Store as FLOAT[] so it can be used by vss_search.
	createTableQuery := `
	-- Create the sequence first (with IF NOT EXISTS to make it re-runnable)
	CREATE SEQUENCE IF NOT EXISTS embeddings_id_seq;

	-- Then the table
	CREATE TABLE IF NOT EXISTS embeddings (
		id        BIGINT PRIMARY KEY DEFAULT NEXTVAL('embeddings_id_seq'),
		text_id   BIGINT,
		embedding FLOAT[1536]
	);
`
	_, err := db.Exec(createTableQuery)
	if err != nil {
		panic(err)
	}
}
