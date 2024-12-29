package data

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func (user User) getUserDb() *sql.DB {

	homeDir, err := getHomeDir()
	if err != nil {
		panic(fmt.Sprintf("did not find home dir for db creation. %s", err))
	}

	path := fmt.Sprintf("%s/.claude/%s.db", homeDir, user.Name)

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		panic(err)
	}

	// Check if the database is new by querying the sqlite_master table
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master").Scan(&count)
	if err != nil {
		panic(err)
	}

	// If the database is new (no tables exist), set it up
	if count == 0 {
		user.setupDb(db)
	}

	return db
}

func (user User) setupDb(db *sql.DB) {
	createHistoryTables(db)
	createContextTable(db)
}

func createContextTable(db *sql.DB) {
	createTableQuery := `
         CREATE TABLE IF NOT EXISTS context (
             id INTEGER PRIMARY KEY AUTOINCREMENT,
             name TEXT
         )
     `

	_, err := db.Exec(createTableQuery)
	if err != nil {
		panic(err)
	}
}

func createHistoryTables(db *sql.DB) {
	createTableQuery := `
         CREATE TABLE IF NOT EXISTS history (
             id INTEGER PRIMARY KEY AUTOINCREMENT,
             context_id INTEGER,
             prompt TEXT,
             response TEXT,
             abreviation TEXT,
             token_count INTEGER
         )
     `
	_, err := db.Exec(createTableQuery)
	if err != nil {
		panic(err)
	}
}

func (user User) InsertContext(context Context) (int64, error) {
	db := user.getUserDb()

	log.Println("inserting context", context.Name, user.Name, user.Id)

	insertQuery := "INSERT INTO context (name) VALUES (?)"
	result, err := db.Exec(insertQuery, context.Name)
	log.Println("result of context insert", result)

	defer db.Close()
	if err != nil {
		log.Println("insert of context failed", err)
		return 0, err
	}

	contextId, err := result.LastInsertId()
	if err != nil {
		log.Println("insert of context failed", err)
		return 0, err
	}

	return contextId, nil
}

func (user User) GetContextById(contextId int64) (Context, error) {
	db := user.getUserDb()
	defer db.Close()

	selectQuery := "SELECT id, name FROM context WHERE id = ?"
	row := db.QueryRow(selectQuery, contextId)

	var context Context
	err := row.Scan(&context.Id, &context.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			// return context, fmt.Errorf("context with ID %d not found", contextId)
		}
		return context, err
	}

	return context, nil
}

func (user User) InsertHistory(history History) (int64, error) {
	db := user.getUserDb()

	insertQuery := "INSERT INTO history (context_id, prompt, response, abreviation, token_count) VALUES (?, ?, ?, ?, ?)"
	result, err := db.Exec(insertQuery, history.ContextId, history.Prompt, history.Response, history.Abbreviation, history.TokenCount)
	if err != nil {
		println(err)
		defer db.Close()
		return 0, err
	}

	historyId, err := result.LastInsertId()
	if err != nil {
		println(err)
		defer db.Close()
		return 0, err
	}

	defer db.Close()
	return historyId, nil
}

func (user User) GetHistoryByContextId(contextId int64, maxCount int) ([]History, error) {
	db := user.getUserDb()
	defer db.Close()

	selectQuery := "SELECT id, context_id, prompt, response, abreviation, token_count FROM history WHERE context_id = ? ORDER BY ID DESC LIMIT ?"
	rows, err := db.Query(selectQuery, contextId, maxCount)
	if err != nil {
		return nil, err
	}

	var histories []History
	for rows.Next() {
		var history History
		err := rows.Scan(&history.Id, &history.ContextId, &history.Prompt, &history.Response, &history.Abbreviation, &history.TokenCount)
		if err != nil {
			return nil, err
		}
		histories = append(histories, history)

	}
	for i, j := 0, len(histories)-1; i < j; i, j = i+1, j-1 {
		histories[i], histories[j] = histories[j], histories[i]
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return histories, nil
}

func (user User) GetContextByName(name string) (*Context, error) {
	db := user.getUserDb()
	defer db.Close()

	selectQuery := "SELECT id, name FROM context WHERE name = ?"
	row := db.QueryRow(selectQuery, name)

	var context Context
	err := row.Scan(&context.Id, &context.Name)

	if err != nil {
		return nil, err
	}

	return &context, err
}

func (user User) GetAllContexts() ([]Context, error) {
	db := user.getUserDb()
	defer db.Close()

	rows, err := db.Query("SELECT id, name FROM context")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contexts []Context
	for rows.Next() {
		var context Context
		err := rows.Scan(&context.Id, &context.Name)
		if err != nil {
			return nil, err
		}
		contexts = append(contexts, context)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return contexts, nil
}

func (user User) DeleteContext(contextId int64) (int64, error) {
	db := user.getUserDb()
	defer db.Close()

	res, err := db.Exec("DELETE FROM context WHERE id = ?", contextId)
	//TODO: should also delete all history connected to the context

	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (user User) DeleteHistory(historyId int64) (int64, error) {
	db := user.getUserDb()
	defer db.Close()

	res, err := db.Exec("DELETE FROM history WHERE id = ?", historyId)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
