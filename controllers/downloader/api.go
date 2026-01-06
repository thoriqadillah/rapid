package downloader

type downloadParams struct {
	Url string `json:"url" validate:"required,url"`
}
