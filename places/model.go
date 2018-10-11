package places

import (
	"github.com/asqwrd/trotter-api/sygic"
	"github.com/asqwrd/trotter-api/triposo"
	"github.com/grokify/html-strip-tags-go"
)

// Place represents the normalized + filtered data for a sygic.Place
type Place struct {
	// These are name overrides
	Sygic_id          string `json:"sygic_id"`
	Image             string `json:"image"`
	Description       string `json:"description"`
	Description_short string `json:"description_short"`

	// These don't
	Name         string            `json:"name"`
	Name_suffix  string            `json:"name_suffix"`
	Parent_ids   []string          `json:"parent_ids"`
	Level        string            `json:"level"`
	Address      string            `json:"address"`
	Phone        string            `json:"phone"`
	Location     sygic.Location    `json:"location"`
	Bounding_box sygic.BoundingBox `json:"bounding_box"`
}

type TriposoPlace struct {
	Id                string           `json:"id"`
	Image             string           `json:"image"`
	Description       string           `json:"description"`
	Description_short string           `json:"description_short"`
	Name              string           `json:"name"`
	Level             string           `json:"level"`
	Location          triposo.Location `json:"location"`
}

func fromSygicPlace(sp *sygic.Place) (p *Place) {
	p = &Place{
		// These have name overrides
		Sygic_id:    sp.ID,
		Image:       sp.Thumbnail_url,
		Description: sp.Perex,

		// These don't
		Name:         sp.Name,
		Name_suffix:  sp.Name_suffix,
		Parent_ids:   sp.Parent_ids,
		Level:        sp.Level,
		Address:      sp.Address,
		Phone:        sp.Phone,
		Location:     sp.Location,
		Bounding_box: sp.Bounding_box,
	}

	return p
}

func FromTriposoPlace(sp *triposo.Place) (p *TriposoPlace) {
	length := len(sp.Images)
	var image = ""
	if length > 0 {
		image = sp.Images[0].Sizes.Medium.Url
	}

	p = &TriposoPlace{
		Id:                sp.Id,
		Image:             image,
		Description:       strip.StripTags(sp.Content.Sections[0].Body),
		Description_short: sp.Snippet,
		Name:              sp.Name,
		Level:             "triposo",
		Location:          triposo.Location{Lat: sp.Coordinates.Latitude, Lng: sp.Coordinates.Longitude},
	}

	return p
}

func FromTriposoPlaces(sourcePlaces []triposo.Place) (internalPlaces []TriposoPlace) {
	internalPlaces = []TriposoPlace{}
	for _, sourcePlace := range sourcePlaces {
		internalPlaces = append(internalPlaces, *FromTriposoPlace(&sourcePlace))
	}

	return internalPlaces
}

// FromSygicPlaces converts a sygic.Place to an internal Place value
func FromSygicPlaces(sourcePlaces []sygic.Place) (internalPlaces []Place) {
	internalPlaces = []Place{}
	for _, sourcePlace := range sourcePlaces {
		internalPlaces = append(internalPlaces, *fromSygicPlace(&sourcePlace))
	}

	return internalPlaces
}
