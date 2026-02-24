package apishortenurlpost

type Request struct {
	URL string `json:"url" validate:"required,url"`
}

type Response struct {
	Result string `json:"result"`
}
