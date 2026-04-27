package domain

type Comic struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Genres      []string `json:"genres"`
	Status      string   `json:"status"`
}

type Chapter struct {
	ID      string `json:"id"`
	ComicID string `json:"comicId"`
	Number  int    `json:"number"`
	Title   string `json:"title"`
}
