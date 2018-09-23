package places

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"github.com/asqwrd/trotter-api/sygic"
	"github.com/gorilla/mux"
)

// Controller unifies the Place controller methods
type Controller struct {
	// Repository Repository
}

// GetContinent aggregates continent data from sygic API
func (c *Controller) GetContinent(w http.ResponseWriter, r *http.Request) {
	routeVars := mux.Vars(r)
	parentID := routeVars["continentID"]

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

	repr := map[string][]Place{
		"points_of_interest": FromSygicPlaces(placesToSee),
		"popular_cities":     FromSygicPlaces(popularCities),
		"all_countries":      FromSygicPlaces(allCountries),
		"popular_countries":  FromSygicPlaces(allCountries[:10]),
	}

	data, _ := json.Marshal(repr)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
	return
}
