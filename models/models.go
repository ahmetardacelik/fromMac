package models

type Artist struct {
	ID        string
	Name      string
	Popularity int
	Followers struct {
		Total int
	}
	Genres []string
}

type TopArtistsResponse struct {
	Items []Artist
}
