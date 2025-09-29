package data

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type PostgresHistoryRepository struct {
	db   *sql.DB
	User User
}

func (postgresHistoryRepository *PostgresHistoryRepository) Init(connectionString string) error {
	db, err := sql.Open("postgres", connectionString)
	log.Println("setting up database", db)
	if err != nil {
		return err
	}

	err = db.Ping()
	if err != nil {
		return err
	}

	postgresHistoryRepository.db = db

	return nil
}

func (r *PostgresHistoryRepository) GetContextById(contextId int64) (Context, error) {
	var context Context
	err := r.db.QueryRow("SELECT id, name, user_id FROM context WHERE id = $1 AND user_id = $2", contextId, r.User.Id).
		Scan(&context.Id, &context.Name, &context.UserId, &context.SystemPrompt)
	if err != nil {
		return Context{}, err
	}
	return context, nil
}

func (r *PostgresHistoryRepository) InsertHistory(history History) (int64, error) {
	var id int64
	err := r.db.QueryRow("INSERT INTO history (context_id, prompt, response, abbreviation, token_count, user_id, created) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
		history.ContextId, history.Prompt, history.Response, history.Abbreviation, history.TokenCount, history.UserId, time.Now()).
		Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *PostgresHistoryRepository) InsertContext(context Context) (int64, error) {
	var id int64
	err := r.db.QueryRow("INSERT INTO context (name, user_id, system_prompt) VALUES ($1, $2, $3) RETURNING id",
		context.Name, context.UserId, context.SystemPrompt).
		Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *PostgresHistoryRepository) GetHistoryByContextId(contextId int64, maxCount int) ([]History, error) {

	// log.Printf("\nSELECT id, context_id, prompt, response, abbreviation, token_count, user_id, created FROM history WHERE context_id = %s AND user_id = %s ORDER BY created DESC LIMIT %s \n", contextId, r.User.Id, maxCount)

	rows, err := r.db.Query("SELECT id, context_id, prompt, response, abbreviation, token_count, user_id, created FROM history WHERE context_id = $1 AND user_id = $2 ORDER BY created DESC LIMIT $3",
		contextId, r.User.Id, maxCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []History
	for rows.Next() {
		log.Println("row in history response")
		var h History
		err := rows.Scan(&h.Id, &h.ContextId, &h.Prompt, &h.Response, &h.Abbreviation, &h.TokenCount, &h.UserId, &h.Created)
		if err != nil {
			log.Println("error parsing history response", err)
			return nil, err
		}
		histories = append(histories, h)
	}
	return histories, nil
}

func (r *PostgresHistoryRepository) GetContextByName(name string) (*Context, error) {
	var context Context
	err := r.db.QueryRow("SELECT id, name, user_id FROM context WHERE name = $1 AND user_id = $2", name, r.User.Id).
		Scan(&context.Id, &context.Name, &context.UserId, &context.SystemPrompt)
	if err != nil {
		log.Println("err selecting context", err)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &context, nil
}

func (r *PostgresHistoryRepository) GetAllContexts() ([]Context, error) {
	rows, err := r.db.Query("SELECT id, name, user_id FROM context WHERE user_id = $1", r.User.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contexts []Context
	for rows.Next() {
		var c Context
		err := rows.Scan(&c.Id, &c.Name, &c.UserId, &c.SystemPrompt)
		if err != nil {
			return nil, err
		}
		contexts = append(contexts, c)
	}
	return contexts, nil
}

func (r *PostgresHistoryRepository) DeleteContext(contextId int64) (int64, error) {
	result, err := r.db.Exec("DELETE FROM context WHERE id = $1 AND user_id = $2", contextId, r.User.Id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *PostgresHistoryRepository) DeleteHistory(historyId int64) (int64, error) {
	result, err := r.db.Exec("DELETE FROM history WHERE id = $1 AND user_id = $2", historyId, r.User.Id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// User CRUD operations

func (r *PostgresHistoryRepository) CreateUser(user User) (int, error) {
	var id int

	log.Println("inserting user", user)
	err := r.db.QueryRow("INSERT INTO users (name, email, slack_id) VALUES ($1, $2, $3) RETURNING id",
		user.Name, user.Email, user.SlackId).
		Scan(&id)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func (r *PostgresHistoryRepository) GetUserById(id string) (*User, error) {
	var user User
	err := r.db.QueryRow("SELECT id, name, email, slack_id FROM users WHERE id = $1", id).
		Scan(&user.Id, &user.Name, &user.Email, &user.SlackId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *PostgresHistoryRepository) UpdateUser(user User) error {
	_, err := r.db.Exec("UPDATE users SET name = $1, email = $2, slack_id = $3 WHERE id = $4",
		user.Name, user.Email, user.SlackId, user.Id)
	return err
}

func (r *PostgresHistoryRepository) DeleteUser(id string) error {
	_, err := r.db.Exec("DELETE FROM users WHERE id = $1", id)
	return err
}

func (r *PostgresHistoryRepository) GetUserByEmail(email string) (*User, error) {
	var user User
	err := r.db.QueryRow("SELECT id, name, email, slack_id FROM users WHERE email = $1", email).
		Scan(&user.Id, &user.Name, &user.Email, &user.SlackId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *PostgresHistoryRepository) GetUserBySlackId(slackId string) (*User, error) {
	var user User
	err := r.db.QueryRow("SELECT id, name, email, slack_id FROM users WHERE slack_id = $1", slackId).
		Scan(&user.Id, &user.Name, &user.Email, &user.SlackId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *PostgresHistoryRepository) UpdateSystemPrompt(contextId int64, systemPrompt string) error {
	_, err := r.db.Exec("UPDATE contexts SET system_prompt = $1 WHERE id = $2",
		systemPrompt, contextId)
	return err
}
