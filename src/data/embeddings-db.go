package data

type EmbeddingMatch struct {
	Id        int64
	TextId    int64
	Text      string
	Distance  float64
	Reference string
}

type EmbeddingsStore interface {
	FindMatches(embedding string) ([]EmbeddingMatch, error)
	InsertEmbedding(text string, embedding string, reference string) (int64, error)
}
