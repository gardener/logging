package e2e

type SearchResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Value []any `json:"value"`
		} `json:"result"`
	} `json:"data"`
}
