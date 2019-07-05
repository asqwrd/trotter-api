package places

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"regexp"
	"strings"

	"github.com/asqwrd/trotter-api/sygic"
	"github.com/asqwrd/trotter-api/triposo"
	"github.com/generaltso/vibrant"
	strip "github.com/grokify/html-strip-tags-go"
	"googlemaps.github.io/maps"
)

var GoogleApi = "AIzaSyDjkQw21rnh9QfJIh2YD-Fl4NEteIBn7L8"

// Place represents the normalized + filtered data for a sygic.Place
type Place struct {
	// These are name overrides
	Sygic_id          string `json:"sygic_id"`
	Id                string `json:"id"`
	Image             string `json:"image"`
	Description       string `json:"description"`
	Description_short string `json:"description_short,omitempty"`

	// These don't
	Name          string            `json:"name"`
	Original_name string            `json:"original_name"`
	Name_suffix   string            `json:"name_suffix,omitempty"`
	Parent_ids    []string          `json:"parent_ids,omitempty"`
	Level         string            `json:"level,omitempty"`
	Address       string            `json:"address,omitempty"`
	Phone         string            `json:"phone,omitempty"`
	Location      sygic.Location    `json:"location"`
	Bounding_box  sygic.BoundingBox `json:"bounding_box"`
	Colors        Colors            `json:"colors"`
	Color         interface{}       `json:"color"`
	Tags          []triposo.Tags    `json:"tags"`
}

type PlaceChannel struct {
	Places interface{}
	Index  int
	Error  error
}

type ColorChannel struct {
	Colors Colors
	Index  int
	Error  error
}

type InternalPlaceChannel struct {
	Place triposo.InternalPlace
	Index int
	Error error
}

type Colors struct {
	Vibrant      string
	Muted        string
	LightVibrant string
	LightMuted   string
	DarkVibrant  string
	DarkMuted    string
}

func FromSygicPlace(sp *sygic.Place) (p *Place) {
	name := sp.Name
	if name == "Czechia" {
		name = "Czech Republic"
	}
	if name == "Ireland" {
		name = "Republic of Ireland"
	}
	p = &Place{
		// These have name overrides
		Sygic_id:    sp.ID,
		Id:          sp.ID,
		Image:       sp.Thumbnail_url,
		Description: sp.Perex,

		// These don't
		Name:          name,
		Original_name: sp.Original_name,
		Name_suffix:   sp.Name_suffix,
		Parent_ids:    sp.Parent_ids,
		Level:         sp.Level,
		Address:       sp.Address,
		Phone:         sp.Phone,
		Location:      sp.Location,
		Bounding_box:  sp.Bounding_box,
	}

	return p
}

//Init Google Maps Client
func InitGoogle() (*maps.Client, error) {
	googleClient, err := maps.NewClient(maps.WithAPIKey(GoogleApi))
	return googleClient, err
}

func FromSygicPlaceDetail(sp *sygic.PlaceDetail) (p *Place) {
	re := regexp.MustCompile(`\{[^\]]*?\}`)
	name := sp.Name
	if name == "Czechia" {
		name = "Czech Republic"
	}
	if name == "Ireland" {
		name = "Republic of Ireland"
	}
	image := ""
	if len(sp.Main_media.Media) > 0 {
		image = re.ReplaceAllString(sp.Main_media.Media[0].Url_template, "1200x800")
	}
	p = &Place{
		// These have name overrides
		Sygic_id:    sp.Id,
		Id:          sp.Id,
		Image:       image,
		Description: sp.Perex,

		// These don't
		Name:          name,
		Original_name: sp.Original_name,
		Location:      sp.Location,
		Bounding_box:  sp.Bounding_box,
	}

	return p
}

func FromGooglePlaceSearch(sp maps.PlacesSearchResult, level string) (p triposo.InternalPlace) {
	length := len(sp.Photos)
	var image = ""
	if length > 0 {
		image = "https://maps.googleapis.com/maps/api/place/photo?maxwidth=1280&photoreference=" + sp.Photos[0].PhotoReference + "&key=" + GoogleApi
	}
	var images []triposo.Image

	for i := 0; i < len(sp.Photos); i++ {
		images = append(images, triposo.Image{
			Sizes: triposo.ImageSizes{
				Medium: triposo.ImageSize{
					Url: "https://maps.googleapis.com/maps/api/place/photo?maxwidth=1280&photoreference=" + sp.Photos[i].PhotoReference + "&key=" + GoogleApi,
				},
				Original: triposo.ImageSize{
					Url: "https://maps.googleapis.com/maps/api/place/photo?maxwidth=1280&photoreference=" + sp.Photos[i].PhotoReference + "&key=" + GoogleApi,
				},
			},
		})
	}
	description := ""
	if len(sp.FormattedAddress) > 0 {
		description = strip.StripTags(sp.FormattedAddress)
	}
	if len(level) == 0 {
		level = "poi"
	}

	var hours string
	var openNow bool
	if sp.OpeningHours != nil && sp.OpeningHours.WeekdayText != nil {
		hours = strings.Join(sp.OpeningHours.WeekdayText, "\n")

	}

	if sp.OpeningHours != nil && sp.OpeningHours.OpenNow != nil {
		openNow = *sp.OpeningHours.OpenNow
	}

	var properties = []triposo.Property{
		triposo.Property{
			Ordinal: 0,
			Value:   sp.FormattedAddress,
			Name:    "Address",
			Key:     "address",
		},
	}

	if len(hours) > 0 {
		properties = append(properties, triposo.Property{
			Ordinal: 1,
			Value:   hours,
			Name:    "Hours",
			Key:     "hours",
		})
	}

	var vicinity = ""
	if len(sp.Vicinity) > 0 {
		vicinity = "Near " + sp.Vicinity
	}
	var gplace = true

	p = triposo.InternalPlace{
		ID:               sp.PlaceID,
		Type:             level,
		Image:            image,
		Images:           images,
		Description:      description,
		DescriptionShort: vicinity,
		Name:             sp.Name,
		Level:            level,
		Location:         triposo.Location{Lat: sp.Geometry.Location.Lat, Lng: sp.Geometry.Location.Lng},
		Score:            sp.Rating * 2,
		OpeningHours:     &triposo.OpeningHours{OpenNow: openNow},
		Properties:       properties,
		GooglePlace: &gplace,
	}

	return p
}

func FromGooglePlace(sp maps.PlaceDetailsResult, level string) (p triposo.InternalPlace) {
	length := len(sp.Photos)
	var image = ""
	if length > 0 {
		image = "https://maps.googleapis.com/maps/api/place/photo?maxwidth=1280&photoreference=" + sp.Photos[0].PhotoReference + "&key=" + GoogleApi
	}
	var images []triposo.Image

	for i := 0; i < len(sp.Photos); i++ {
		images = append(images, triposo.Image{
			Sizes: triposo.ImageSizes{
				Medium: triposo.ImageSize{
					Url: "https://maps.googleapis.com/maps/api/place/photo?maxwidth=1280&photoreference=" + sp.Photos[i].PhotoReference + "&key=" + GoogleApi,
				},
				Original: triposo.ImageSize{
					Url: "https://maps.googleapis.com/maps/api/place/photo?maxwidth=1280&photoreference=" + sp.Photos[i].PhotoReference + "&key=" + GoogleApi,
				},
			},
		})
	}
	description := ""
	if len(sp.AdrAddress) > 0 {
		description = strip.StripTags(sp.AdrAddress)
	}
	if len(sp.FormattedAddress) > 0 {
		description = strip.StripTags(sp.FormattedAddress)
	}
	if len(level) == 0 {
		level = "poi"
	}

	var hours string
	var openNow bool
	if sp.OpeningHours != nil && sp.OpeningHours.WeekdayText != nil {
		hours = strings.Join(sp.OpeningHours.WeekdayText, "\n")

	}

	if sp.OpeningHours != nil && sp.OpeningHours.OpenNow != nil {
		openNow = *sp.OpeningHours.OpenNow
	}

	var properties = []triposo.Property{
		triposo.Property{
			Ordinal: 0,
			Value:   sp.FormattedAddress,
			Name:    "Address",
			Key:     "address",
		},
	}

	if len(sp.InternationalPhoneNumber) > 0 {
		properties = append(properties, triposo.Property{
			Ordinal: 1,
			Value:   sp.InternationalPhoneNumber,
			Name:    "Phone",
			Key:     "phone",
		})
	}

	if len(hours) > 0 {
		properties = append(properties, triposo.Property{
			Ordinal: 1,
			Value:   hours,
			Name:    "Hours",
			Key:     "hours",
		})
	}

	var vicinity = ""
	if len(sp.Vicinity) > 0 {
		vicinity = "Near " + sp.Vicinity
	}

	p = triposo.InternalPlace{
		ID:               sp.PlaceID,
		Type:             level,
		Image:            image,
		Images:           images,
		Description:      description,
		DescriptionShort: vicinity,
		Name:             sp.Name,
		Level:            level,
		Location:         triposo.Location{Lat: sp.Geometry.Location.Lat, Lng: sp.Geometry.Location.Lng},
		Score:            sp.Rating * 2,
		OpeningHours:     &triposo.OpeningHours{OpenNow: openNow},
		Properties:       properties,
	}

	return p
}

// FromTriposoPlace function that converts response
func FromTriposoPlace(sp triposo.Place, level string, thumbnail ...bool) (p triposo.InternalPlace) {
	length := len(sp.Images)
	var image = ""
	var image_hd = ""
	var areaIndex = 0
	var area = 0

	if length > 0 {
		for i := 0; i < length; i++ {
			var a = sp.Images[i].Sizes.Original.Width * sp.Images[i].Sizes.Original.Height
			if area < a {
				area = a
				areaIndex = i
			}
		}

		if len(thumbnail) > 0 && thumbnail[0] == true {
			image = sp.Images[areaIndex].Sizes.Medium.Url
			image_hd = sp.Images[areaIndex].Sizes.Original.Url
		} else {
			image = sp.Images[areaIndex].Sizes.Original.Url
			image_hd = sp.Images[areaIndex].Sizes.Original.Url
		}
	}

	description := ""
	if len(sp.Content.Sections) > 0 {
		description = strip.StripTags(sp.Content.Sections[0].Body)
	}
	if len(level) == 0 {
		level = sp.Type
	}

	p = triposo.InternalPlace{
		ID:               sp.ID,
		Type:             sp.Type,
		Image:            image,
		ImageHD:          image_hd,
		Images:           sp.Images,
		Description:      description,
		DescriptionShort: sp.Snippet,
		Name:             sp.Name,
		Level:            level,
		Location:         triposo.Location{Lat: sp.Coordinates.Latitude, Lng: sp.Coordinates.Longitude},
		BestFor:          sp.BestFor,
		PriceTier:        sp.PriceTier,
		FacebookID:       sp.FacebookID,
		FoursquareID:     sp.FoursquareID,
		TripadvisorID:    sp.TripadvisorID,
		GooglePlaceID:    sp.GooglePlaceID,
		BookingInfo:      sp.BookingInfo,
		Score:            sp.Score,
		OpeningHours:     sp.OpeningHours,
		Properties:       sp.Properties,
		ParentID:         sp.ParentID,
		CountryID:        sp.CountryID,
		LocationID:       sp.LocationID,
		Trigram:          sp.Trigram,
		Tags:             sp.Tags,
	}

	return p
}

//FromTriposoPlaces func
func FromTriposoPlaces(sourcePlaces []triposo.Place, level string) (internalPlaces []triposo.InternalPlace) {
	internalPlaces = []triposo.InternalPlace{}
	for _, sourcePlace := range sourcePlaces {
		internalPlaces = append(internalPlaces, FromTriposoPlace(sourcePlace, level, true))
	}

	return internalPlaces
}

// FromSygicPlaces converts a sygic.Place to an internal Place value
func FromSygicPlaces(sourcePlaces []sygic.Place) (internalPlaces []Place) {
	internalPlaces = []Place{}
	for _, sourcePlace := range sourcePlaces {
		internalPlaces = append(internalPlaces, *FromSygicPlace(&sourcePlace))
	}

	return internalPlaces
}

func GetColor(url string) (*Colors, error) {

	res, err := http.Get(url)
	if err != nil {
		fmt.Println("here")
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
			colors.Vibrant = swatch.Color.RGBHex()
		case "Muted":
			colors.Muted = swatch.Color.RGBHex()
		case "LightVibrant":
			colors.LightVibrant = swatch.Color.RGBHex()
		case "LightMuted":
			colors.LightMuted = swatch.Color.RGBHex()
		case "DarkVibrant":
			colors.DarkVibrant = swatch.Color.RGBHex()
		case "DarkMuted":
			colors.DarkMuted = swatch.Color.RGBHex()
		}
	}

	return &colors, nil
}
