package embeddings

import (
	"fmt"
	"os"

	commontypes "owl/common_types"
	"owl/data"
	"owl/logger"
	embeddings_model "owl/models/open-ai-embedings"
	"owl/services"

	"github.com/fatih/color"
)

// Config bundles settings for embedding generation and search.
// Backend: "sqlite" (default) or "duckdb".
// Store: when true, InsertEmbedding will be called; when false, the embedding
// response is returned on ResponseChannel for downstream use (e.g. search).
// Reference: optional reference stored alongside the text (e.g. file path).
// ChunkPath: if set, file will be chunked via services.ChunkMarkdown and each
// chunk embedded+stored.
// SearchQuery: if set, query will be embedded and used to FindMatches.
// Prompt: used when generating a single embedding (no chunking).
// NOTE: only one of (ChunkPath, SearchQuery, Prompt) should be used at a time.

type Config struct {
	DBName           string
	EmbeddingsDBName string
	Store            bool
	Reference        string
	ChunkPath        string
	SearchQuery      string
	Prompt           string
}

func defaultedDBName() string {
	db := os.Getenv("OWL_LOCAL_DATABASE")
	if db == "" {
		return "owl"
	}
	return db
}

func defaultedEmbeddingsDBName() string {
	db := os.Getenv("OWL_LOCAL_EMBEDDINGS_DATABASE")
	if db == "" {
		return "owl_embeddings"
	}
	return db
}

func storeForBackend(backend, embeddingsDB string) data.EmbeddingsStore {
	switch backend {
	// case "sqlite":
	// 	return &data.EmbeddingsDatabase{Name: embeddingsDB}
	case "duckdb":
		fallthrough
	default:
		return &data.DuckDbEmbeddingsDatabase{Name: embeddingsDB}
	}
}

// Run executes embedding generation, chunk+store, or search depending on cfg.
func Run(cfg Config) ([]data.EmbeddingMatch, error) {
	if cfg.DBName == "" {
		cfg.DBName = defaultedDBName()
	}
	if cfg.EmbeddingsDBName == "" {
		cfg.EmbeddingsDBName = defaultedEmbeddingsDBName()
	}

	store := storeForBackend("duckdb", cfg.EmbeddingsDBName)

	rh := &ResponseHandler{Db: store, Store: cfg.Store, Reference: cfg.Reference}
	// Only needed for search (we need the embedding back)
	if !cfg.Store {
		rh.Init()
	}

	user := data.User{Name: &cfg.DBName}
	model := embeddings_model.OpenAiEmbeddingsModel{ResponseHandler: rh}

	switch {
	case cfg.ChunkPath != "":
		bytes, err := os.ReadFile(cfg.ChunkPath)
		if err != nil {
			return nil, fmt.Errorf("could not read file: %w", err)
		}
		chunks := services.ChunkMarkdown(string(bytes))
		logger.Screen(fmt.Sprintf("Chunked document into %d pieces", len(chunks)), color.RGB(150, 250, 150))

		rh.Reference = cfg.ChunkPath
		for i, chunkStr := range chunks {
			logger.Screen(fmt.Sprintf("Processing chunk %d/%d (size: %d chars)", i+1, len(chunks), len(chunkStr)), color.RGB(150, 150, 250))
			services.AwaitedQuery(chunkStr, &model, user, 0, nil, &commontypes.PayloadModifiers{}, "embeddings")
		}
		return nil, nil

	case cfg.SearchQuery != "":
		services.AwaitedQuery(cfg.SearchQuery, &model, user, 0, nil, &commontypes.PayloadModifiers{}, "embeddings")
		embedding := <-rh.ResponseChannel

		matches, err := store.FindMatches(embedding)
		if err != nil {
			return nil, err
		}

		for i, match := range matches {
			logger.Screen(fmt.Sprintf("\n\nindex of match: %d, distance: %f", i, match.Distance), color.RGB(250, 250, 150))
			logger.Screen(fmt.Sprintf("\nDistance: %f\nReference: %s\nText: %s\n", match.Distance, match.Reference, match.Text), color.RGB(150, 150, 150))
		}
		return matches, nil

	case cfg.Prompt != "":
		services.AwaitedQuery(cfg.Prompt, &model, user, 0, nil, &commontypes.PayloadModifiers{}, "embeddings")
		return nil, nil
	default:
		return nil, fmt.Errorf("no embeddings action specified")
	}
}
