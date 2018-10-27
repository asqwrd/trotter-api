package sygic

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type placesResponse struct {
	Status_code int
	Data        placesData
}

type placesData struct {
	Places []Place
}

type Location struct {
	Lat float32 `json:"lat"`
	Lng float32 `json:"lng"`
}

type BoundingBox struct {
	South float32 `json:"south"`
	West  float32 `json:"west"`
	North float32 `json:"north"`
	East  float32 `json:"east"`
}

type Object struct {
	data interface{}
}

type Place struct {
	// These names get overridden
	ID            string
	Thumbnail_url string
	Perex         string

	// These don't
	Name          string
	Original_name string
	Name_suffix   string
	Parent_ids    []string
	Level         string
	Address       string
	Phone         string
	Location      Location
	Bounding_box  BoundingBox
	Color         string
	Visa          Object
	Plugs         Object
}

const baseSygicAPI = "https://api.sygictravelapi.com/1.1/en/places/"

var sygicAPIKey = os.Getenv("SYGIC_API_KEY")

func request(parentID string, limit int, query *url.Values) (*http.Response, error) {
	client := http.Client{Timeout: time.Second * 5}

	req, err := http.NewRequest(http.MethodGet, baseSygicAPI+"list", nil)
	if err != nil {
		return nil, err
	}

	var q *url.Values
	if query == nil {
		args := req.URL.Query()
		q = &args
	} else {
		q = query
	}

	q.Set("parents", parentID)
	q.Set("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("x-api-key", sygicAPIKey)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func GetPlaces(parentID string, limit int, query *url.Values) ([]Place, error) {
	res, err := request(parentID, limit, query)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Sygic API.")
	}

	resp := &placesResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Server experienced an error while parsing Sygic API response.")
	}

	return resp.Data.Places, nil
}

type placeResponse struct {
	Data placeData
}

type placeData struct {
	Place PlaceDetail
}

type PlaceDetail struct {
	Id            string
	Main_media    mainMedia
	Name          string
	Original_name string
	Perex         string
	Location      Location
	Bounding_box  BoundingBox
}

type mainMedia struct {
	Usage usage
	Media []media
}

type usage struct {
	Square        string `json:"square"`
	Video_preview string `json:"video_preview"`
	Portrait      string `json:"portrait"`
	Landscape     string `json:"landscape"`
}

type media struct {
	Url          string
	Url_template string
}
type SygicChannel struct {
	Places []Place
	Index  int
	Error  error
}

func GetPlace(placeID string) (*PlaceDetail, error) {
	client := http.Client{Timeout: time.Second * 5}

	req, err := http.NewRequest(http.MethodGet, baseSygicAPI+placeID, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Sygic API.")
	}

	req.Header.Set("x-api-key", sygicAPIKey)

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Sygic API.")
	}

	resp := &placeResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Server experienced an error while parsing Sygic API response.")
	}

	return &resp.Data.Place, nil
}

func GetCountry(count string) (*[]Place, error) {
	client := http.Client{Timeout: time.Second * 5}

	req, err := http.NewRequest(http.MethodGet, baseSygicAPI+"list?level=country&limit="+count, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	req.Header.Set("x-api-key", sygicAPIKey)

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

	return &resp.Data.Places, nil
}
