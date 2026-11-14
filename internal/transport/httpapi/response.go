package httpapi

type Response struct {
	Provider  string `json:"provider"`
	Key       string `json:"key"`
	Value     string `json:"value_base64"`
	FetchedAt string `json:"fetched_at"`
	Stale     string `json:"stale"`
}
