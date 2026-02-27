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
	// When getting by ID, we unarchive it
	_, _ = r.db.Exec("UPDATE context SET archived = 0 WHERE id = $1 AND user_id = $2", contextId, r.User.Id)

	var context Context
	var archived int
	err := r.db.QueryRow("SELECT id, name, user_id, system_prompt, COALESCE(preferred_model, 'sonnet'), archived FROM context WHERE id = $1 AND user_id = $2", contextId, r.User.Id).
		Scan(&context.Id, &context.Name, &context.UserId, &context.SystemPrompt, &context.PreferredModel, &archived)
	if err != nil {
		return Context{}, err
	}
	context.Archived = archived == 1
	return context, nil
}

func (r *PostgresHistoryRepository) InsertHistory(history History) (int64, error) {
	var id int64
	err := r.db.QueryRow("INSERT INTO history (context_id, prompt, response, abbreviation, token_count, user_id, created, response_content, model) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id",
		history.ContextId, history.Prompt, history.Response, history.Abbreviation, history.TokenCount, history.UserId, time.Now(), history.ResponseContent, history.Model).
		Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *PostgresHistoryRepository) InsertContext(context Context) (int64, error) {
	var id int64
	err := r.db.QueryRow("INSERT INTO context (name, user_id, system_prompt, preferred_model) VALUES ($1, $2, $3, $4) RETURNING id",
		context.Name, context.UserId, context.SystemPrompt, context.PreferredModel).
		Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *PostgresHistoryRepository) GetHistoryByContextId(contextId int64, maxCount int) ([]History, error) {
	rows, err := r.db.Query("SELECT id, context_id, prompt, response, abbreviation, token_count, user_id, created, COALESCE(model, 'sonnet'), archived FROM history WHERE context_id = $1 AND user_id = $2 ORDER BY created DESC LIMIT $3",
		contextId, r.User.Id, maxCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []History
	for rows.Next() {
		var h History
		var archived int
		err := rows.Scan(&h.Id, &h.ContextId, &h.Prompt, &h.Response, &h.Abbreviation, &h.TokenCount, &h.UserId, &h.Created, &h.Model, &archived)
		if err != nil {
			log.Println("error parsing history response", err)
			return nil, err
		}
		h.Archived = archived == 1
		histories = append(histories, h)
	}
	return histories, nil
}

func (r *PostgresHistoryRepository) GetContextByName(name string) (*Context, error) {
	// When getting by name, we unarchive it
	_, _ = r.db.Exec("UPDATE context SET archived = 0 WHERE name = $1 AND user_id = $2", name, r.User.Id)

	var context Context
	var archived int
	err := r.db.QueryRow("SELECT id, name, user_id, system_prompt, COALESCE(preferred_model, 'sonnet'), archived FROM context WHERE name = $1 AND user_id = $2", name, r.User.Id).
		Scan(&context.Id, &context.Name, &context.UserId, &context.SystemPrompt, &context.PreferredModel, &archived)
	if err != nil {
		log.Println("err selecting context", err)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	context.Archived = archived == 1
	return &context, nil
}

func (r *PostgresHistoryRepository) GetAllContexts() ([]Context, error) {
	rows, err := r.db.Query("SELECT id, name, user_id, system_prompt, COALESCE(preferred_model, 'sonnet'), archived FROM context WHERE user_id = $1", r.User.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contexts []Context
	for rows.Next() {
		var c Context
		var archived int
		err := rows.Scan(&c.Id, &c.Name, &c.UserId, &c.SystemPrompt, &c.PreferredModel, &archived)
		if err != nil {
			return nil, err
		}
		c.Archived = archived == 1
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

func (r *PostgresHistoryRepository) UpdateSystemPrompt(contextId int64, systemPrompt string) error {
	_, err := r.db.Exec("UPDATE contexts SET system_prompt = $1 WHERE id = $2",
		systemPrompt, contextId)
	return err
}

func (r *PostgresHistoryRepository) UpdatePreferredModel(contextId int64, model string) error {
	_, err := r.db.Exec("UPDATE contexts SET preferred_model = $1 WHERE id = $2",
		model, contextId)
	return err
}

func (r *PostgresHistoryRepository) ArchiveContext(contextId int64, archived bool) error {
	val := 0
	if archived {
		val = 1
	}
	_, err := r.db.Exec("UPDATE context SET archived = $1 WHERE id = $2 AND user_id = $3", val, contextId, r.User.Id)
	return err
}

func (r *PostgresHistoryRepository) ArchiveHistory(historyId int64, archived bool) error {
	val := 0
	if archived {
		val = 1
	}
	_, err := r.db.Exec("UPDATE history SET archived = $1 WHERE id = $2 AND user_id = $3", val, historyId, r.User.Id)
	return err
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
