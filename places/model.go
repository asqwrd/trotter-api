package places

import "github.com/asqwrd/trotter-api/sygic"

// Place represents the normalized + filtered data for a sygic.Place
type Place struct {
	// These are name overrides
	Sygic_id    string `json:"sygic_id"`
	Image       string `json:"image"`
	Description string `json:"description"`
	Description_short string `json:"description_short"`

	// These don't
	Name        string         `json:"name"`
	Name_suffix string         `json:"name_suffix"`
	Parent_ids  []string       `json:"parent_ids"`
	Level       string         `json:"level"`
	Address     string         `json:"address"`
	Phone       string         `json:"phone"`
	Location    sygic.Location `json:"location"`
  
  Color					string						`json:"color"`
  Visa	 				sygic.Object			`json:"visa"`
  Plugs 				sygic.Object			`json:"plugs"`
  Bounding_box 	sygic.BoundingBox	`json:"bounding_box"`
}

func fromSygicPlace(sp *sygic.Place) (p *Place) {
	p = &Place{
		// These have name overrides
		Sygic_id:    sp.ID,
		Image:       sp.Thumbnail_url,
		Description: sp.Perex,

		// These don't
		Name:        		sp.Name,
		Name_suffix: 		sp.Name_suffix,
		Parent_ids:  		sp.Parent_ids,
		Level:       		sp.Level,
		Address:     		sp.Address,
		Phone:       		sp.Phone,
		Location:    		sp.Location,
		Bounding_box:   sp.Bounding_box,
		Color:    			sp.Color,
		Visa:    				sp.Visa,
		Plugs:    			sp.Plugs,
	}

	return p
}

// FromSygicPlaces converts a sygic.Place to an internal Place value
func FromSygicPlaces(sourcePlaces []sygic.Place) (internalPlaces []Place) {
	internalPlaces = []Place{}
	for _, sourcePlace := range sourcePlaces {
		internalPlaces = append(internalPlaces, *fromSygicPlace(&sourcePlace))
	}

	return internalPlaces
}
