package attachments

type Asset struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Size         int64   `json:"size"`
	ContentType  string  `json:"content_type"`
	Href         string  `json:"href"`
	OriginalName *string `json:"original_name"`
}
