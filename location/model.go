package location

import (
	"github.com/asqwrd/trotter-api/sygic"
	"github.com/asqwrd/trotter-api/triposo"
)

type Location struct {
	// These are name overrides / added data
	Title    string `json:"title"`
	Selected bool   `json:"selected"`

	// These are direct
	BoundingBox sygic.BoundingBox `json:"bounding_box"`
	Lat         float32           `json:"lat"`
	Lng         float32           `json:"lng"`
}

func fromSygicPlace(p *sygic.Place) Location {
	return Location{
		// Overrides
		Title:    p.Name,
		Selected: false,

		// Direct
		Lat:         p.Location.Lat,
		Lng:         p.Location.Lng,
		BoundingBox: p.Bounding_box,
	}
}

func fromTriposoPlace(p *triposo.InternalPlace) Location {
	return Location{
		// Overrides
		Title:    p.Name,
		Selected: false,

		// Direct
		Lat: p.Location.Lat,
		Lng: p.Location.Lng,
	}
}

func FromSygicPlaces(places []sygic.Place) []Location {
	locations := []Location{}

	for _, sygPlace := range places {
		locations = append(locations, fromSygicPlace(&sygPlace))
	}

	return locations
}

func FromTriposoPlaces(places []triposo.InternalPlace) []Location {
	locations := []Location{}

	for _, tripPlace := range places {
		locations = append(locations, fromTriposoPlace(&tripPlace))
	}

	return locations
}
