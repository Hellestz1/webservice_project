package domain

type Comic struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Author      string   `json:"author"`
	Country     string   `json:"country"`
	Description string   `json:"description"`
	Genres      []string `json:"genres"`
	BookType    string   `json:"bookType"`
	Status      string   `json:"status"`
}

type ComicSearchFilters struct {
	Query     string
	YearFrom  int
	YearTo    int
	Category  string
	AgeRating string
	BookType  string
	SortBy    string
	Order     string
	Author    string
	Country   string
	Limit     int
	Page      int
}

type Chapter struct {
	ID      string `json:"id"`
	ComicID string `json:"comicId"`
	Number  int    `json:"number"`
	Title   string `json:"title"`
}
