package triposo

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

type placesResponse struct {
	Results []Place
}

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Object struct {
	data interface{}
}

type Sections struct {
	Body string `json:"body"`
}

type Content struct {
	Sections []Sections `json:"sections"`
}

type MediumSize struct {
	Url string `json:"url"`
}

type ImageSizes struct {
	Medium MediumSize `json:"medium"`
}

//Image struct
type Image struct {
	OwnerURL string     `json:"owner_url"`
	Sizes    ImageSizes `json:"sizes"`
}

//Place struct
type Place struct {
	Name          string        `json:"name"`
	ID            string        `json:"id"`
	Type          string        `json:"type"`
	Coordinates   Coordinates   `json:"coordinates"`
	Content       Content       `json:"content"`
	Images        []Image       `json:"images"`
	Snippet       string        `json:"snippet"`
	Score         float32       `json:"score"`
	LocationID    string        `json:"location_id"`
	FacebookID    string        `json:"facebook_id"`
	FoursquareID  string        `json:"foursquare_id"`
	GooglePlaceID string        `json:"google_place_id"`
	TripadvisorID string        `json:"tripadvisor_id"`
	PriceTier     int           `json:"price_tier"`
	BookingInfo   *BookingInfo  `json:"booking_info,omitempty"`
	BestFor       []BestFor     `json:"best_for,omitempty"`
	Intro         string        `json:"intro"`
	OpeningHours  *OpeningHours `json:"opening_hours,omitempty"`
	Properties    []Property    `json:"properties,omitempty"`
	ParentID      string        `json:"parent_id,omitempty"`
	CountryID     string        `json:"country_id,omitempty"`
	Trigram       float32       `json:"trigram"`
	GooglePlace   bool          `json:"google_place" firestore:"googe_place"`
}

//BestFor struct
type BestFor struct {
	Label     string `json:"label"`
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
	Snippet   string `json:"snippet"`
}

//OpeningHours struct
type OpeningHours struct {
	Days    *TimeRangesByDay `json:"days,omitempty"`
	OpenNow bool             `json:"open_now" firestore:"open_now"`
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
	Ordinal int    `json:"ordinal"`
	Value   string `json:"value"`
	Name    string `json:"name"`
	Key     string `json:"key"`
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
	ID               string        `json:"id"`
	Type             string        `json:"type"`
	Image            string        `json:"image,omitempty"`
	Description      string        `json:"description" json:"intro"`
	DescriptionShort string        `json:"description_short,omitempty"`
	Name             string        `json:"name"`
	Level            string        `json:"level"`
	Location         Location      `json:"location"`
	LocationID       string        `json:"location_id"`
	FacebookID       string        `json:"facebook_id,omitempty"`
	FoursquareID     string        `json:"foursquare_id,omitempty"`
	GooglePlaceID    string        `json:"google_place_id,omitempty"`
	TripadvisorID    string        `json:"tripadvisor_id,omitempty"`
	PriceTier        int           `json:"price_tier,omitempty"`
	BookingInfo      *BookingInfo  `json:"booking_info,omitempty"`
	BestFor          []BestFor     `json:"best_for"`
	Images           []Image       `json:"images"`
	Score            float32       `json:"score"`
	OpeningHours     *OpeningHours `json:"opening_hours,omitempty"`
	Properties       []Property    `json:"properties"`
	ParentID         string        `json:"parent_id,omitempty"`
	ParentName       string        `json:"parent_name,omitempty"`
	CountryName      string        `json:"country_name,omitempty"`
	CountryID        string        `json:"country_id,omitempty"`
	Trigram          float32       `json:"trigram"`
	GooglePlace      *bool         `json:"google_place" firestore:"googe_place"`
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
	Error  error
}

const baseTriposoAPI = "https://www.triposo.com/api/20181213/"

const TRIPOSO_ACCOUNT = "2ZWR5MHH"
const TRIPOSO_TOKEN = "yan4ujbhzepr66ttsqxiqwcl38k3lx0w"

func GetPlaceByName(name string) (*PoiInfo, error) {
	client := http.Client{Timeout: time.Second * 10}

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
		url = baseTriposoAPI + "poi.json?location_id=" + location_id + "&fields=intro,images,location_id,id,content,opening_hours,coordinates,snippet,score,facebook_id,attribution,best_for,properties,price_tier,name,booking_info&annotate=trigram:" + query + "&trigram=>=0.3&account=" + TRIPOSO_ACCOUNT + "&token=" + TRIPOSO_TOKEN
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
	client := http.Client{Timeout: time.Second * 10}

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

func GetPoiFromLocation(id string, count string, tag_labels string, index int) (*[]Place, error) {

	client := http.Client{Timeout: time.Second * 10}
	url := baseTriposoAPI + "poi.json?location_id=" + id + "&count=" + count + "&fields=id,name,coordinates,facebook_id,location_id,opening_hours,snippet,content,best_for,properties,images&account=" + TRIPOSO_ACCOUNT + "&token=" + TRIPOSO_TOKEN
	if len(tag_labels) > 0 {
		url = baseTriposoAPI + "poi.json?location_id=" + id + "&tag_labels=" + tag_labels + "&count=" + count + "&fields=id,name,coordinates,location_id,opening_hours,snippet,content,best_for,properties,images&account=" + TRIPOSO_ACCOUNT + "&token=" + TRIPOSO_TOKEN
	}

	//fmt.Println(url)

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

	//fmt.Println(res)

	resp := &placesResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Triposo API response.")
	}

	return &resp.Results, nil

}

func GetLocation(id string) (*[]Place, error) {
	client := http.Client{Timeout: time.Second * 10}

	req, err := http.NewRequest(http.MethodGet, baseTriposoAPI+"location.json?id="+id+"&order_by=-score&fields=coordinates,parent_id,images,content,name,id,snippet,type&account="+TRIPOSO_ACCOUNT+"&token="+TRIPOSO_TOKEN, nil)
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
	client := http.Client{Timeout: time.Second * 10}

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
	client := http.Client{Timeout: time.Second * 10}

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
