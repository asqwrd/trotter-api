package places

import (
	"net/http"
	"net/url"

	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/sygic"
	"github.com/gorilla/mux"
)

// GetContinent aggregates continent data from sygic API
func GetContinent(w http.ResponseWriter, r *http.Request) {
	routeVars := mux.Vars(r)
	parentID := routeVars["continentID"]

	placesToSeeArgs := &url.Values{}
	placesToSeeArgs.Set("categories", "sightseeing")
	placesToSee, err := sygic.GetPlaces(parentID, "poi", 10, placesToSeeArgs)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	popularCitiesArgs := &url.Values{"rating": []string{".0005:"}}
	popularCities, err := sygic.GetPlaces(parentID, "city", 10, popularCitiesArgs)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	allCountries, err := sygic.GetPlaces(parentID, "country", 50, nil)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	responseData := map[string][]Place{
		"points_of_interest": FromSygicPlaces(placesToSee),
		"popular_cities":     FromSygicPlaces(popularCities),
		"all_countries":      FromSygicPlaces(allCountries),
		"popular_countries":  FromSygicPlaces(allCountries[:10]),
	}

	response.Write(w, responseData, http.StatusOK)
	return
}
