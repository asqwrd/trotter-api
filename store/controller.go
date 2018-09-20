package store

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"github.com/asqwrd/trotter-api/places"
	"github.com/asqwrd/trotter-api/sygic"
)

type Controller struct {
	// Repository Repository
}

const parentID = "continent:1"

// Index GET /
func (c *Controller) Index(w http.ResponseWriter, r *http.Request) {
	placesToSeeArgs := &url.Values{}
	placesToSeeArgs.Set("categories", "sightseeing")

	placesToSee, err := sygic.GetPlaces(parentID, "poi", 10, placesToSeeArgs)
	if err != nil {
		log.Println(err)
		// TODO: return an error
	}

	popularCitiesArgs := &url.Values{"rating": []string{".0005:"}}
	popularCities, err := sygic.GetPlaces(parentID, "city", 10, popularCitiesArgs)
	if err != nil {
		log.Println(err)
		// TODO: return an error
	}

	allCountries, err := sygic.GetPlaces(parentID, "country", 50, nil)
	if err != nil {
		log.Println(err)
		// TODO: return an error
	}

	repr := map[string][]places.Place{
		"points_of_interest": places.PlacesFromSygicPlaces(placesToSee),
		"popular_cities":     places.PlacesFromSygicPlaces(popularCities),
		"all_countries":      places.PlacesFromSygicPlaces(allCountries),
		"popular_countries":  places.PlacesFromSygicPlaces(allCountries[:10]),
	}

	data, _ := json.Marshal(repr)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
	return
}
