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
	Lat float32 `json:"lat"`
	Lng float32 `json:"lng"`
}

type Coordinates struct {
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
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

type Image struct {
	Owner_url string     `json:"owner_url"`
	Sizes     ImageSizes `json:"sizes"`
}

type Place struct {
	Name            string        `json:"name"`
	Id              string        `json:"id"`
	Coordinates     Coordinates   `json:"coordinates"`
	Content         Content       `json:"content"`
	Images          []Image       `json:"images"`
	Snippet         string        `json:"snippet"`
	Score           float32       `json:"score"`
	Location_id     string        `json:"location_id"`
	Facebook_id     string        `json:"facebook_id"`
	Foursquare_id   string        `json:"foursquare_id"`
	Google_place_id string        `json:"google_place_id"`
	Tripadvisor_id  string        `json:"tripadvisor_id"`
	Price_tier      int           `json:"price_tier"`
	Booking_info    *Booking_info `json:"booking_info,omitempty"`
	Best_for        []BestFor     `json:"best_for,omitempty"`
	Intro           string        `json:"intro"`
	Opening_hours   *OpeningHours `json:"opening_hours,omitempty"`
	Properties      []Property    `json:"properties,omitempty"`
	Parent_Id       string        `json:"parent_id,omitempty"`
	Country_Id      string        `json:"country_id,omitempty"`
}

type BestFor struct {
	Label      string `json:"label"`
	Name       string `json:"name"`
	Short_name string `json:"short_name"`
	Snippet    string `json:"snippet"`
}

type OpeningHours struct {
	Days *TimeRangesByDay `json:"days,omitempty"`
}

type TimeRangesByDay struct {
	Mon []TimeRange `json:"mon"`
	Tue []TimeRange `json:"tue"`
	Wed []TimeRange `json:"wed"`
	Thu []TimeRange `json:"thu"`
	Fri []TimeRange `json:"fri"`
	Sat []TimeRange `json:"sat,"`
	Sun []TimeRange `json:"sun"`
}

type TimeRange struct {
	End   DayTime `json:"end"`
	Start DayTime `json:"start"`
}

type DayTime struct {
	Hour   int `json:"hour,omitempty"`
	Minute int `json:"minute"`
}

type Property struct {
	Ordinal int    `json:"ordinal"`
	Value   string `json:"value"`
	Name    string `json:"name"`
	Key     string `json:"key"`
}

type Booking_info struct {
	Price             *Price `json:"price,omitempty"`
	Vendor            string `json:"vendor,omitempty"`
	Vendor_object_id  string `json:"vendor_object_id,omitempty"`
	Vendor_object_url string `json:"vendor_object_url,omitempty"`
}

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

type InternalPlace struct {
	Id                string        `json:"id"`
	Image             string        `json:"image,omitempty"`
	Description       string        `json:"description" json:"intro"`
	Description_short string        `json:"description_short,omitempty"`
	Name              string        `json:"name"`
	Level             string        `json:"level"`
	Location          Location      `json:"location"`
	Facebook_id       string        `json:"facebook_id,omitempty"`
	Foursquare_id     string        `json:"foursquare_id,omitempty"`
	Google_place_id   string        `json:"google_place_id,omitempty"`
	Tripadvisor_id    string        `json:"tripadvisor_id,omitempty"`
	Price_tier        int           `json:"price_tier,omitempty"`
	Booking_info      *Booking_info `json:"booking_info,omitempty"`
	Best_for          []BestFor     `json:"best_for"`
	Images            []Image       `json:"images"`
	Score             float32       `json:"score"`
	Opening_hours     *OpeningHours `json:"opening_hours,omitempty"`
	Properties        []Property    `json:"properties"`
	Parent_Id         string        `json:"parent_id,omitempty"`
	Parent_Name       string        `json:"parent_name,omitempty"`
	Country_Id        string        `json:"country_id,omitempty"`
}

type PoiInfo struct {
	Country_id string  `json:"country_id"`
	Id         string  `json:"id"`
	Trigram    float32 `json:"trigram"`
}

type TriposoChannel struct {
	Places []Place
	Index  int
	Error  error
}

const baseTriposoAPI = "https://www.triposo.com/api/20180627/"

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

func Search(query string, typeParam string) (*[]Place, error) {
	client := http.Client{Timeout: time.Second * 20}
	url := baseTriposoAPI + "location.json?type=" + typeParam + "&order_by=-trigram&fields=name,parent_id,score,images,id,type,coordinates,country_id,snippet,content,properties,intro&annotate=trigram:" + query + "&trigram=>=0.3&account=" + TRIPOSO_ACCOUNT + "&token=" + TRIPOSO_TOKEN
	if typeParam == "poi" {
		url = baseTriposoAPI + "poi.json?fields=google_place_id,intro,tripadvisor_id,images,location_id,id,content,opening_hours,coordinates,snippet,score,facebook_id,attribution,best_for,properties,price_tier,name,foursquare_id,booking_info&annotate=trigram:" + query + "&trigram=>=0.3&account=" + TRIPOSO_ACCOUNT + "&token=" + TRIPOSO_TOKEN
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
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

	resp := &placeResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Triposo API response.")
	}
	//fmt.Println(resp.Results)
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
	req, err := http.NewRequest(http.MethodGet, baseTriposoAPI+"poi.json?id="+id+"&fields=google_place_id,images,id,name,booking_info,best_for,facebook_id,opening_hours,score,tripadvisor_id,content,foursquare_id,price_tier,intro,snippet,properties,coordinates&account="+TRIPOSO_ACCOUNT+"&token="+TRIPOSO_TOKEN, nil)
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
	url := baseTriposoAPI + "poi.json?location_id=" + id + "&count=" + count + "&fields=google_place_id,id,name,coordinates,tripadvisor_id,facebook_id,location_id,opening_hours,foursquare_id,snippet,content,best_for,properties,images&account=" + TRIPOSO_ACCOUNT + "&token=" + TRIPOSO_TOKEN
	if len(tag_labels) > 0 {
		url = baseTriposoAPI + "poi.json?location_id=" + id + "&tag_labels=" + tag_labels + "&count=" + count + "&fields=google_place_id,id,name,coordinates,tripadvisor_id,facebook_id,location_id,opening_hours,foursquare_id,snippet,content,best_for,properties,images&account=" + TRIPOSO_ACCOUNT + "&token=" + TRIPOSO_TOKEN
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

	req, err := http.NewRequest(http.MethodGet, baseTriposoAPI+"location.json?id="+id+"&order_by=-score&fields=coordinates,parent_id,images,content,name,id,snippet&account="+TRIPOSO_ACCOUNT+"&token="+TRIPOSO_TOKEN, nil)
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

	req, err := http.NewRequest(http.MethodGet, baseTriposoAPI+"location.json?type="+type_id+"&count="+count+"&order_by=-score&fields=coordinates,parent_id,images,content,name,id,score,snippet&account="+TRIPOSO_ACCOUNT+"&token="+TRIPOSO_TOKEN, nil)
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
