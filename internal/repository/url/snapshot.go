package url

type Snapshot struct {
	URLs        map[string]string   `json:"urls"`
	UserURLs    map[string][]string `json:"user_urls"`
	DeletedURLs map[string]bool     `json:"deleted_urls"`
	Creators    map[string]string   `json:"creators"`
}
