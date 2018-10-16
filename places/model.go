package places

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"regexp"

	"github.com/asqwrd/trotter-api/sygic"
	"github.com/asqwrd/trotter-api/triposo"
	"github.com/generaltso/vibrant"
	"github.com/grokify/html-strip-tags-go"
)

// Place represents the normalized + filtered data for a sygic.Place
type Place struct {
	// These are name overrides
	Sygic_id          string `json:"sygic_id"`
	Image             string `json:"image"`
	Description       string `json:"description"`
	Description_short string `json:"description_short,omitempty"`

	// These don't
	Name         string            `json:"name"`
	Name_suffix  string            `json:"name_suffix,omitempty"`
	Parent_ids   []string          `json:"parent_ids,omitempty"`
	Level        string            `json:"level,omitempty"`
	Address      string            `json:"address,omitempty"`
	Phone        string            `json:"phone,omitempty"`
	Location     sygic.Location    `json:"location"`
	Bounding_box sygic.BoundingBox `json:"bounding_box"`
	Colors       Colors            `json:"colors"`
	Color        interface{}       `json:"color"`
}

type PlaceChannel struct {
	Places interface{}
	Index  int
	Error  error
}

type Colors struct {
	Vibrant      string
	Muted        string
	LightVibrant string
	LightMuted   string
	DarkVibrant  string
	DarkMuted    string
}

func FromSygicPlace(sp *sygic.Place, colors Colors) (p *Place) {
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
		Colors:       colors,
	}

	return p
}

func FromSygicPlaceDetail(sp *sygic.PlaceDetail, colors Colors) (p *Place) {
	re := regexp.MustCompile(`\{[^\]]*?\}`)
	p = &Place{
		// These have name overrides
		Sygic_id:    sp.Id,
		Image:       re.ReplaceAllString(sp.Main_media.Media[0].Url_template, "1200x800"),
		Description: sp.Perex,

		// These don't
		Name:         sp.Name,
		Location:     sp.Location,
		Bounding_box: sp.Bounding_box,
		Colors:       colors,
	}

	return p
}

func FromTriposoPlace(sp *triposo.Place) (p *triposo.InternalPlace) {
	length := len(sp.Images)
	var image = ""
	if length > 0 {
		image = sp.Images[0].Sizes.Medium.Url
	}

	p = &triposo.InternalPlace{
		Id:                sp.Id,
		Image:             image,
		Images:            sp.Images,
		Description:       strip.StripTags(sp.Content.Sections[0].Body),
		Description_short: sp.Snippet,
		Name:              sp.Name,
		Level:             "triposo",
		Location:          triposo.Location{Lat: sp.Coordinates.Latitude, Lng: sp.Coordinates.Longitude},
		Best_for:          sp.Best_for,
		Price_tier:        sp.Price_tier,
		Facebook_id:       sp.Facebook_id,
		Foursquare_id:     sp.Foursquare_id,
		Tripadvisor_id:    sp.Tripadvisor_id,
		Google_place_id:   sp.Google_place_id,
		Booking_info:      sp.Booking_info,
		Score:             sp.Score,
		Opening_hours:     sp.Opening_hours,
	}

	return p
}

func FromTriposoPlaces(sourcePlaces []triposo.Place) (internalPlaces []triposo.InternalPlace) {
	internalPlaces = []triposo.InternalPlace{}
	for _, sourcePlace := range sourcePlaces {
		internalPlaces = append(internalPlaces, *FromTriposoPlace(&sourcePlace))
	}

	return internalPlaces
}

// FromSygicPlaces converts a sygic.Place to an internal Place value
func FromSygicPlaces(sourcePlaces []sygic.Place) (internalPlaces []Place) {
	internalPlaces = []Place{}
	var colors Colors
	for _, sourcePlace := range sourcePlaces {
		internalPlaces = append(internalPlaces, *FromSygicPlace(&sourcePlace, colors))
	}

	return internalPlaces
}

func GetColor(url string) (*Colors, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	img, _, err := image.Decode(res.Body)
	if err != nil {
		return nil, err
	}
	palette, err := vibrant.NewPaletteFromImage(img)
	if err != nil {
		return nil, err
	}
	var colors Colors
	for name, swatch := range palette.ExtractAwesome() {
		switch name {
		case "Vibrant":
			colors.Vibrant = swatch.Color.String()
		case "Muted":
			colors.Muted = swatch.Color.String()
		case "LightVibrant":
			colors.LightVibrant = swatch.Color.String()
		case "LightMuted":
			colors.LightMuted = swatch.Color.String()
		case "DarkVibrant":
			colors.DarkVibrant = swatch.Color.String()
		case "DarkMuted":
			colors.DarkMuted = swatch.Color.String()
		}
	}

	return &colors, nil
}
