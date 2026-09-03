package gmeta

type Paging struct {
	Page   int   `json:"page"`
	Limit  int   `json:"limit"`
	Offset int   `json:"-"`
	Total  int64 `json:"total"`
}
