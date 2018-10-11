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
	Name        string      `json:"name"`
	Id          string      `json:"id"`
	Coordinates Coordinates `json:"coordinates"`
	Content     Content     `json:"content"`
	Images      []Image     `json:"images"`
	Snippet     string      `json:"snippet"`
	Score       float32     `json:"score"`
	Location_id string      `json:"location_id"`
	Parent_id   string      `json:"parent_id"`
}

type placeResponse struct {
	Results []Place
}

type poiInfoResponse struct {
	Results []PoiInfo
}

type InternalPlace struct {
	Id                string   `json:"id"`
	Image             string   `json:"image"`
	Description       string   `json:"description"`
	Description_short string   `json:"description_short"`
	Name              string   `json:"name"`
	Level             string   `json:"level"`
	Location          Location `json:"location"`
}

type PoiInfo struct {
	Country_id string  `json:"country_id"`
	Id         string  `json:"id"`
	Trigram    float32 `json:"trigram"`
}

type TriposoChannel struct {
	Places []Place
	Index  int
	Error error
}

const baseTriposoAPI = "https://www.triposo.com/api/20180627/"

const TRIPOSO_ACCOUNT = "2ZWR5MHH"
const TRIPOSO_TOKEN = "yan4ujbhzepr66ttsqxiqwcl38k3lx0w"

func GetPlaceByName(name string) (*PoiInfo, error) {
	client := http.Client{Timeout: time.Second * 5}

	req, err := http.NewRequest(http.MethodGet, baseTriposoAPI+"location.json?order_by=-trigram&count=1&fields=id,country_id&annotate=trigram:"+name+"&trigram=>=0.3&account="+TRIPOSO_ACCOUNT+"&token="+TRIPOSO_TOKEN, nil)
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

	resp := &poiInfoResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Sygic API response.")
	}

	return &resp.Results[0], nil
}

func GetDestination(id string, count string) (*[]Place, error) {
	client := http.Client{Timeout: time.Second * 5}

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
		return nil, errors.New("Server experienced an error while parsing Sygic API response.")
	}

	return &resp.Results, nil
}

func GetPoiFromLocation(id string, count string, tag_labels string, index int) (*[]Place, error) {

	client := http.Client{Timeout: time.Second * 5}

	req, err := http.NewRequest(http.MethodGet, baseTriposoAPI+"poi.json?location_id="+id+"&tag_labels="+tag_labels+"&count="+count+"&fields=google_place_id,id,name,coordinates,tripadvisor_id,facebook_id,location_id,opening_hours,foursquare_id,snippet,content,images&account="+TRIPOSO_ACCOUNT+"&token="+TRIPOSO_TOKEN, nil)
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
		return nil, errors.New("Server experienced an error while parsing Sygic API response.")
	}

	return &resp.Results, nil

}

func GetCity(id string) (*[]Place, error) {
	client := http.Client{Timeout: time.Second * 5}

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
		return nil, errors.New("Server experienced an error while parsing Sygic API response.")
	}

	return &resp.Results, nil

}

func GetLocationType(type_id string, count string) (*[]Place, error){
	client := http.Client{Timeout: time.Second * 5}

	req, err := http.NewRequest(http.MethodGet, baseTriposoAPI+"location.json?type="+type_id+"&count="+count+"&order_by=-score&fields=coordinates,parent_id,images,content,name,id,snippet&account="+TRIPOSO_ACCOUNT+"&token="+TRIPOSO_TOKEN, nil)
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
		return nil, errors.New("Server experienced an error while parsing Sygic API response.")
	}

	return &resp.Results, nil
}
