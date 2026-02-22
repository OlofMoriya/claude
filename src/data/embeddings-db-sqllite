package data

import (
	"database/sql"
	"fmt"
	"owl/logger"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

// SQLite (sqlite-vec) implementation of EmbeddingsStore.
// Note: sqlite-vec expects a vector value for MATCH. In this codebase embeddings
// are currently passed around as a string; ensure the string is in the format
// expected by sqlite-vec (e.g. JSON array) when inserting/searching.

type EmbeddingsDatabase struct {
	Name string
}

func (edb *EmbeddingsDatabase) getUserDb() *sql.DB {
	homeDir, err := getHomeDir()
	if err != nil {
		panic(fmt.Sprintf("did not find home dir for db creation. %s", err))
	}

	path := fmt.Sprintf("%s/.owl/%s.db", homeDir, edb.Name)

	sqlite_vec.Auto()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		panic(err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master").Scan(&count)
	if err != nil {
		panic(err)
	}

	if count == 0 {
		edb.setupDb(db)
	}

	return db
}

func (edb *EmbeddingsDatabase) setupDb(db *sql.DB) {
	createEmbeddingsTable(db)
	createTextTable(db)
}

func (edb *EmbeddingsDatabase) FindMatches(embedding string) ([]EmbeddingMatch, error) {
	db := edb.getUserDb()
	defer db.Close()

	logger.Debug.Println("searching embedding (sqlite)")

	selectQuery := `
		SELECT e.id, e.text_id, t.content, e.distance, t.reference
		FROM embeddings e
		JOIN texts t ON e.text_id = t.id
		WHERE e.embedding MATCH ? AND k = 3 LIMIT 3;`

	rows, err := db.Query(selectQuery, embedding)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("error while searching for embeddings: %s", err.Error())
		}
		return nil, err
	}
	defer rows.Close()

	var matches []EmbeddingMatch
	for rows.Next() {
		var match EmbeddingMatch
		err := rows.Scan(&match.Id, &match.TextId, &match.Text, &match.Distance, &match.Reference)
		if err != nil {
			return nil, err
		}
		matches = append(matches, match)
	}

	return matches, nil
}

func (edb *EmbeddingsDatabase) InsertEmbedding(text string, embedding string, reference string) (int64, error) {
	db := edb.getUserDb()
	defer db.Close()

	logger.Debug.Println("inserting embedding (sqlite)")

	insertTextQuery := "INSERT INTO texts (content, reference) VALUES (?, ?)"
	textResult, err := db.Exec(insertTextQuery, text, reference)
	if err != nil {
		logger.Debug.Println("insert of text failed", err)
		return 0, err
	}

	textId, err := textResult.LastInsertId()
	if err != nil {
		logger.Debug.Println("failed to get text id", err)
		return 0, err
	}

	insertQuery := "INSERT INTO embeddings (text_id, embedding) VALUES (?, ?)"
	result, err := db.Exec(insertQuery, textId, embedding)
	if err != nil {
		logger.Debug.Println("insert of embeddings failed", err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		logger.Debug.Println("insert of embeddings failed", err)
		return 0, err
	}

	return id, nil
}

func createTextTable(db *sql.DB) {
	createTableQuery := `
         CREATE TABLE IF NOT EXISTS texts (
             id INTEGER PRIMARY KEY AUTOINCREMENT,
             content TEXT NOT NULL,
             reference TEXT NOT NULL
         );
     `

	_, err := db.Exec(createTableQuery)
	if err != nil {
		panic(err)
	}
}

func createEmbeddingsTable(db *sql.DB) {
	createTableQuery := `
		create virtual table embeddings using vec0(
		  id INTEGER PRIMARY KEY AUTOINCREMENT,
		  text_id INTEGER,
		  embedding float[1536]
		);
     `

	_, err := db.Exec(createTableQuery)
	if err != nil {
		panic(err)
	}
}
