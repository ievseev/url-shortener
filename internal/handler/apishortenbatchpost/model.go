package apishortenbatchpost

type Request struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type Response struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
