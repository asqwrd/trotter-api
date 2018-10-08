package triposo

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

type placesResponse struct {
	Results        placesData
}

type placesData struct {
	Places []Place
}

type Location struct {
	Lat float32 `json:"lat"`
	Lng float32 `json:"lng"`
}

type Coordinates struct {
	Latitude	float32	`json:"latitude"`
	Longitude float32	`json:"longitude"`
};

type Object struct {
	data interface{}
}

type Sections struct {
	Body	string	`json:"body"`
}

type Content struct {
	Sections	[]Sections `json:"sections"`
}

type MediumSize struct {
	Url			string	`json:"url"`
}

type ImageSizes struct {
	Medium		MediumSize	`json:"medium"`
}

type Images struct {
	Owner_url		string			`json:"owner_url"`
	Sizes				ImageSizes	`json:"sizes"`
}


type Place struct {
	// These names get overridden
	ID            	string
	Location_id			string

	
	// These don't
  Name								string				`json:"name"`
	Opening_hours				string 				`json:"opening_hours"`
  Coordinates					Coordinates		`json:"coordinates"`
  Intro								string				`json:"intro"`
  Content							Content				`json:"content"`
  Images							[]Images			`json:"images"`
  Facebook_id 				string				`json:"facebook_id"`
  Foursquare_id				string				`json:"foursquare_id"`
  Google_place_id			string				`json:"google_place_id"`
  Snippet							string				`json:"snippet"`
  Score								float32				`json:"score"`

}

type placeResponse struct {
	Results []PlaceDetail
}

type poiInfoResponse struct {
	Results []PoiInfo
}

type placeData struct {
	Place PlaceDetail
}

type PlaceDetail struct {
	

	// These don't
  Name								string				`json:"name"`
	Opening_hours				string 				`json:"opening_hours"`
  Coordinates					Coordinates		`json:"coordinates"`
  Intro								string				`json:"intro"`
  Content							Content				`json:"content"`
  Images							[]Images			`json:"images"`
  Facebook_id 				string				`json:"facebook_id"`
  Foursquare_id				string				`json:"foursquare_id"`
  Google_place_id			string				`json:"google_place_id"`
  Score								float32				`json:"score"`
	Id									string				`json:"id"`
	Parent_id						string				`json:"parent_id"`
}

type PoiInfo struct {
	Country_id								string				`json:"country_id"`
	Id												string 				`json:"id"`
  Trigram										float32				`json:"trigram"`
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

func GetDestination(id string, count string) (*[]PlaceDetail, error) {
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

