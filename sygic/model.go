package sygic

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type placeResponse struct {
	Status_code int
	Data        placesData
}

type placesData struct {
	Places []Place
}

type Place struct {
	// These names get overridden
	ID            string
	Thumbnail_url string
	Perex         string

	// These don't
	Name        string
	Name_suffix string
	Parent_ids  []string
	Level       string
	Address     string
	Phone       string

	// TODO: location:curr.location,
}

const baseSygicAPI = "https://api.sygictravelapi.com/1.1/en/places/list"

func request(parentID string, level string, limit int, query *url.Values) (*http.Response, error) {
	client := http.Client{Timeout: time.Second * 5}

	req, err := http.NewRequest(http.MethodGet, baseSygicAPI, nil)
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
	q.Set("level", level)
	q.Set("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	req.Header.Set("x-api-key", "6SdxevLXN2aviv5g67sac2aySsawGYvJ6UcTmvWE")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func GetPlaces(parentID string, level string, limit int, query *url.Values) ([]Place, error) {
	res, err := request(parentID, level, limit, query)
	if err != nil {
		return nil, err
	}

	resp := &placeResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		return nil, err
	}

	return resp.Data.Places, nil
}
