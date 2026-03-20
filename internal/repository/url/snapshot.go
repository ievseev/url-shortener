package url

type Snapshot struct {
	URLs     map[string]string   `json:"urls"`
	UserURLs map[string][]string `json:"user_urls"`
}
