package places

import "github.com/asqwrd/trotter-api/sygic"

type Place struct {
	// These are name overrides
	Sygic_id    string `json:"sygic_id"`
	Image       string `json:"image"`
	Description string `json:"description"`

	// These don't
	Name        string   `json:"name"`
	Name_suffix string   `json:"name_suffix"`
	Parent_ids  []string `json:"parent_ids"`
	Level       string   `json:"level"`
	Address     string   `json:"address"`
	Phone       string   `json:"phone"`
}

func PlaceFromSygicPlace(sp *sygic.Place) (p *Place) {
	p = &Place{
		// These have name overrides
		Sygic_id:    sp.ID,
		Image:       sp.Thumbnail_url,
		Description: sp.Perex,

		// These don't
		Name:        sp.Name,
		Name_suffix: sp.Name_suffix,
		Parent_ids:  sp.Parent_ids,
		Level:       sp.Level,
		Address:     sp.Address,
		Phone:       sp.Phone,
	}

	return p
}

func PlacesFromSygicPlaces(sourcePlaces []sygic.Place) (internalPlaces []Place) {
	internalPlaces = []Place{}
	for _, sourcePlace := range sourcePlaces {
		internalPlaces = append(internalPlaces, *PlaceFromSygicPlace(&sourcePlace))
	}

	return internalPlaces
}
