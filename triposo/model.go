package triposo

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"googlemaps.github.io/maps"
)

type placesResponse struct {
	Results []Place
	More    bool `json:"more" firestore:"more"`
}

type Location struct {
	Lat float64 `json:"lat" firestore:"lat"`
	Lng float64 `json:"lng" firestore:"lng"`
}

type Coordinates struct {
	Latitude  float64 `json:"latitude" firestore:"latitude"`
	Longitude float64 `json:"longitude" firestore:"longitude"`
}

type Object struct {
	data interface{}
}

type Sections struct {
	Body string `json:"body" firestore:"body"`
}

type Content struct {
	Sections []Sections `json:"sections" firestore:"sections"`
}

type ImageSize struct {
	Url    string `json:"url" firestore:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Bytes  int    `json:"bytes"`
}

type ImageSizes struct {
	Medium    ImageSize `json:"medium" firestore:"medium"`
	Original  ImageSize `json:"original" firestore:"original"`
	Thumbnail ImageSize `json:"thumbnail" firestore:"thumbnail"`
}

//Image struct
type Image struct {
	OwnerURL string     `json:"owner_url" firestore:"owner_url"`
	SourceID string     `json:"source_id" firestore:"source_id"`
	Sizes    ImageSizes `json:"sizes" firestore:"sizes"`
}

//Place struct
type Place struct {
	Name          string        `json:"name" firestore:"name"`
	ID            string        `json:"id" firestore:"id"`
	Type          string        `json:"type" firestore:"type"`
	Coordinates   Coordinates   `json:"coordinates" firestore:"coordinates"`
	Content       Content       `json:"content" firestore:"content"`
	Images        []Image       `json:"images" firestore:"images"`
	Snippet       string        `json:"snippet" firestore:"snippet"`
	Score         float32       `json:"score" firestore:"score"`
	LocationID    string        `json:"location_id" firestore:"location_id"`
	FacebookID    string        `json:"facebook_id" firestore:"facebook_id"`
	FoursquareID  string        `json:"foursquare_id" firestore:"foursquare_id"`
	GooglePlaceID string        `json:"google_place_id" firestore:"google_place_id"`
	TripadvisorID string        `json:"tripadvisor_id" firestore:"tripadvisor_id"`
	PriceTier     int           `json:"price_tier" firestore:"price_tier"`
	BookingInfo   *BookingInfo  `json:"booking_info,omitempty" firestore:"booking_info"`
	BestFor       []BestFor     `json:"best_for,omitempty" firestore:"best_for"`
	Intro         string        `json:"intro" firestore:"intro"`
	OpeningHours  *OpeningHours `json:"opening_hours,omitempty" firestore:"opening_hours"`
	Properties    []Property    `json:"properties,omitempty" firestore:"properties"`
	ParentID      string        `json:"parent_id,omitempty" firestore:"parent_id"`
	CountryID     string        `json:"country_id,omitempty" firestore:"country_id"`
	Trigram       float32       `json:"trigram" firestore:"trigram"`
	GooglePlace   bool          `json:"google_place" firestore:"googe_place"`
	Tags          []Tags        `json:"tags" firestore:"tags"`
	Color         interface{}   `json:"color,omitempty" firestore:"color,omitempty"`
}

// Tag struct
type Tag struct {
	Name       string `json:"name"`
	ShortName  string `json:"short_name"`
	LocationID string `json:"location_id"`
	Label      string `json:"label"`
}

//Tags Struct
type Tags struct {
	Tags Tag `json:"tag" firestore:"tag"`
}

//BestFor struct
type BestFor struct {
	Label     string `json:"label" firestore:"label"`
	Name      string `json:"name" firestore:"name"`
	ShortName string `json:"short_name" firestore:"short_name"`
	Snippet   string `json:"snippet" firestore:"snippet"`
}

//OpeningHours struct
type OpeningHours struct {
	Days    *TimeRangesByDay `json:"days,omitempty" firestore:"days,omitempty"`
	OpenNow *bool            `json:"open_now,omitempty" firestore:"open_now,omitempty"`
}

//TimeRangesByDay struct
type TimeRangesByDay struct {
	Mon []TimeRange `json:"mon"`
	Tue []TimeRange `json:"tue"`
	Wed []TimeRange `json:"wed"`
	Thu []TimeRange `json:"thu"`
	Fri []TimeRange `json:"fri"`
	Sat []TimeRange `json:"sat,"`
	Sun []TimeRange `json:"sun"`
}

//TimeRange struct
type TimeRange struct {
	End   DayTime `json:"end"`
	Start DayTime `json:"start"`
}

//DayTime struct
type DayTime struct {
	Hour   int `json:"hour,omitempty"`
	Minute int `json:"minute"`
}

//Property struct
type Property struct {
	Ordinal int    `json:"ordinal" firestore:"ordinal"`
	Value   string `json:"value" firestore:"value"`
	Name    string `json:"name" firestore:"name"`
	Key     string `json:"key" firestore:"key"`
}

//BookingInfo struct
type BookingInfo struct {
	Price           *Price `json:"price,omitempty"`
	Vendor          string `json:"vendor,omitempty"`
	VendorObjectID  string `json:"vendor_object_id,omitempty"`
	VendorObjectURL string `json:"vendor_object_url,omitempty"`
}

// Price struct
type Price struct {
	Amount   string `json:"amount,omitempty"`
	Currency string `json:"currency,omitempty"`
}

type placeResponse struct {
	Results []Place
}

type poiInfoResponse struct {
	Results []PoiInfo
}

//InternalPlace struct
type InternalPlace struct {
	ID               string             `json:"id" firestore:"id"`
	Type             string             `json:"type" firestore:"type"`
	Image            string             `json:"image,omitempty" firestore:"image,omitempty"`
	ImageMedium      string             `json:"image_medium,omitempty" firestore:"image_medium,omitempty"`
	ImageHD          string             `json:"image_hd,omitempty" firestore:"image_hd,omitempty"`
	Description      string             `json:"description" json:"intro" firestore:"intro" firestore:"description"`
	DescriptionShort string             `json:"description_short,omitempty" firestore:"description_short,omitempty"`
	Name             string             `json:"name" firestore:"name"`
	Level            string             `json:"level" firestore:"level"`
	Location         Location           `json:"location" firestore:"location"`
	Coordinates      *Coordinates       `json:"coordinates" firestore:"coordinates"`
	LocationID       string             `json:"location_id" firestore:"location_id"`
	FacebookID       string             `json:"facebook_id,omitempty" firestore:"facebook_id,omitempty"`
	FoursquareID     string             `json:"foursquare_id,omitempty" firestore:"foursquare_id,omitempty"`
	GooglePlaceID    string             `json:"google_place_id,omitempty" firestore:"google_place_id,omitempty"`
	TripadvisorID    string             `json:"tripadvisor_id,omitempty" firestore:"tripadvisor_id,omitempty"`
	PriceTier        int                `json:"price_tier,omitempty" firestore:"price_tier,omitempty"`
	BookingInfo      *BookingInfo       `json:"booking_info,omitempty" firestore:"booking_info,omitempty"`
	BestFor          []BestFor          `json:"best_for" firestore:"best_for"`
	Images           []Image            `json:"images" firestore:"images"`
	Score            float32            `json:"score" firestore:"score"`
	OpeningHours     *OpeningHours      `json:"opening_hours,omitempty" firestore:"opening_hours,omitempty"`
	Properties       []Property         `json:"properties" firestore:"properties"`
	ParentID         string             `json:"parent_id,omitempty" firestore:"parent_id,omitempty"`
	ParentName       string             `json:"parent_name,omitempty" firestore:"parent_name,omitempty"`
	CountryName      string             `json:"country_name,omitempty" firestore:"country_name,omitempty"`
	CountryID        string             `json:"country_id,omitempty" firestore:"country_id,omitempty"`
	Trigram          float32            `json:"trigram" firestore:"trigram"`
	GooglePlace      *bool              `json:"google_place" firestore:"googe_place"`
	Tags             []Tags             `json:"tags" firestore:"tags"`
	Reviews          []maps.PlaceReview `json:"reviews" firestore:"reviews"`
	Color            interface{}        `json:"color,omitempty" firestore:"color,omitempty"`
}

//PoiInfo struct
type PoiInfo struct {
	CountryID string  `json:"country_id"`
	ID        string  `json:"id"`
	Trigram   float32 `json:"trigram"`
}

type TriposoChannel struct {
	Places []Place
	Index  int
	More   *bool
	Error  error
}

const baseTriposoAPI = "https://www.triposo.com/api/latest/"

const TRIPOSO_ACCOUNT = "2ZWR5MHH"
const TRIPOSO_TOKEN = "yan4ujbhzepr66ttsqxiqwcl38k3lx0w"

func GetPlaceByName(name string) (*PoiInfo, error) {
	client := http.Client{Timeout: time.Second * 30}

	req, err := http.NewRequest(http.MethodGet, baseTriposoAPI+"location.json?type=country&order_by=-trigram&count=1&fields=id,country_id&annotate=trigram:"+name+"&trigram=>=0.3&account="+TRIPOSO_ACCOUNT+"&token="+TRIPOSO_TOKEN, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}
	//fmt.Println(name)

	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	resp := &poiInfoResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Triposo API response.")
	}
	//fmt.Println(resp.Results)
	return &resp.Results[0], nil
}

func Search(query string, typeParam string, location_id string) (*[]Place, error) {
	client := http.Client{Timeout: time.Second * 20}
	url := baseTriposoAPI + "location.json?type=" + typeParam + "&order_by=-trigram&fields=name,parent_id,score,images,id,type,coordinates,country_id,snippet,content,properties,intro&annotate=trigram:" + query + "&trigram=>=0.3&account=" + TRIPOSO_ACCOUNT + "&token=" + TRIPOSO_TOKEN
	if typeParam == "poi" {
		url = baseTriposoAPI + "poi.json?location_id=" + location_id + "&fields=intro,images,location_id,id,content,opening_hours,coordinates,snippet,score,facebook_id,attribution,best_for,tags,properties,price_tier,name,booking_info&annotate=trigram:" + query + "&trigram=>=0.2&account=" + TRIPOSO_ACCOUNT + "&token=" + TRIPOSO_TOKEN
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	resp := &placeResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Triposo API response.")
	}
	return &resp.Results, nil
}

func GetDestination(id string, count string) (*[]Place, error) {
	client := http.Client{Timeout: time.Second * 30}

	req, err := http.NewRequest(http.MethodGet, baseTriposoAPI+"location.json?part_of="+id+"&order_by=-score&count="+count+"&fields=id,score,parent_id,country_id,intro,name,images,content,coordinates&type=city&account="+TRIPOSO_ACCOUNT+"&token="+TRIPOSO_TOKEN, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	resp := &placeResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Triposo API response.")
	}

	return &resp.Results, nil
}

func GetPoi(id string) (*[]Place, error) {
	client := http.Client{Timeout: time.Second * 10}
	req, err := http.NewRequest(http.MethodGet, baseTriposoAPI+"poi.json?id="+id+"&fields=images,id,name,booking_info,best_for,facebook_id,opening_hours,score,content,price_tier,intro,location_id,snippet,properties,coordinates&account="+TRIPOSO_ACCOUNT+"&token="+TRIPOSO_TOKEN, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	resp := &placeResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Triposo API response.")
	}

	return &resp.Results, nil

}

func GetPoiFromLocation(id string, count string, tag_labels string, index int) (*[]Place, *bool, error) {

	client := http.Client{Timeout: time.Second * 30}
	url := baseTriposoAPI + "poi.json?location_id=" + id + "&count=" + count + "&fields=id,score,name,coordinates,facebook_id,location_id,opening_hours,snippet,content,best_for,properties,images&account=" + TRIPOSO_ACCOUNT + "&token=" + TRIPOSO_TOKEN
	if len(tag_labels) > 0 {
		url = baseTriposoAPI + "poi.json?location_id=" + id + "&tag_labels=" + tag_labels + "&count=" + count + "&fields=id,score,name,coordinates,location_id,opening_hours,snippet,content,best_for,properties,images&account=" + TRIPOSO_ACCOUNT + "&token=" + TRIPOSO_TOKEN
	}

	//fmt.Println(url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println(err)
		return nil, nil, errors.New("Failed to access the Triposo API.")
	}

	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, nil, errors.New("Failed to access the Triposo API.")
	}

	//fmt.Println(res)

	resp := &placesResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, nil, errors.New("Server experienced an error while parsing Triposo API response.")
	}

	return &resp.Results, &resp.More, nil

}

func GetPoiFromLocationPagination(id string, count string, tag_labels string, offset string) (*[]Place, *bool, error) {

	client := http.Client{Timeout: time.Second * 30}
	url := baseTriposoAPI + "poi.json?location_id=" + id + "&tag_labels=" + tag_labels + "&count=" + count + "&offset=" + offset + "&fields=id,score,name,coordinates,location_id,opening_hours,snippet,content,best_for,properties,images&account=" + TRIPOSO_ACCOUNT + "&token=" + TRIPOSO_TOKEN

	//fmt.Println(url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println(err)
		return nil, nil, errors.New("Failed to access the Triposo API.")
	}

	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, nil, errors.New("Failed to access the Triposo API.")
	}

	//fmt.Println(res)

	resp := &placesResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, nil, errors.New("Server experienced an error while parsing Triposo API response.")
	}

	return &resp.Results, &resp.More, nil

}

func GetLocation(id string) (*[]Place, error) {
	client := http.Client{Timeout: time.Second * 30}

	req, err := http.NewRequest(http.MethodGet, baseTriposoAPI+"location.json?id="+id+"&order_by=-score&fields=coordinates,parent_id,country_id,images,content,name,id,snippet,type&account="+TRIPOSO_ACCOUNT+"&token="+TRIPOSO_TOKEN, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	resp := &placesResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Triposo API response.")
	}

	return &resp.Results, nil

}

func GetLocationType(type_id string, count string) (*[]Place, error) {
	client := http.Client{Timeout: time.Second * 30}

	req, err := http.NewRequest(http.MethodGet, baseTriposoAPI+"location.json?type="+type_id+"&count="+count+"&order_by=-score&fields=coordinates,parent_id,country_id,images,content,name,id,score,snippet&account="+TRIPOSO_ACCOUNT+"&token="+TRIPOSO_TOKEN, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	resp := &placesResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Triposo API response.")
	}

	return &resp.Results, nil
}

func GetLocations(count string) (*[]Place, error) {
	client := http.Client{Timeout: time.Second * 30}

	req, err := http.NewRequest(http.MethodGet, baseTriposoAPI+"location.json?count="+count+"&order_by=-score&fields=parent_id,country_id,name,id,score,coordinates,type,images&account="+TRIPOSO_ACCOUNT+"&token="+TRIPOSO_TOKEN, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	resp := &placesResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Triposo API response.")
	}

	return &resp.Results, nil
}
