package places

import (
	"net/http"
	"net/url"

	"github.com/asqwrd/trotter-api/location"
	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/sygic"
	"github.com/asqwrd/trotter-api/triposo"

	"github.com/gorilla/mux"
)

func initializeQueryParams(level string) *url.Values {
	qp := &url.Values{}
	qp.Set("level", level)
	return qp
}

// GetContinent aggregates continent data from sygic API
func GetContinent(w http.ResponseWriter, r *http.Request) {
	routeVars := mux.Vars(r)
	parentID := routeVars["continentID"]

	allCountriesArgs := initializeQueryParams("country")
	allCountries, err := sygic.GetPlaces(parentID, 60, allCountriesArgs)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	popular_countries := FromSygicPlaces(allCountries[:5]);
	popularCities := []triposo.PlaceDetail{}
	for _, country := range popular_countries {
		place, err := triposo.GetPlaceByName(country.Name)
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		city, err := triposo.GetDestination(place.Id,"2")
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		popularCities = append(popularCities, *city...)
	}
	
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}



	responseData := map[string]interface{}{
		"popular_cities":     popularCities,
		"all_countries":      FromSygicPlaces(allCountries),
	}

	response.Write(w, responseData, http.StatusOK)
	return
}

func GetCountry(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status": "coming soon",
	}

	response.Write(w, data, http.StatusNotFound)
	return
}

func GetCity(w http.ResponseWriter, r *http.Request) {
	cityID := mux.Vars(r)["cityID"]

	placesToSeeArgs := initializeQueryParams("poi")
	placesToSeeArgs.Set("categories", "sightseeing")
	placesToSee, err := sygic.GetPlaces(cityID, 20, placesToSeeArgs)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	discoverArgs := initializeQueryParams("poi")
	discoverArgs.Set("categories", "discovering")
	discoverPlaces, err := sygic.GetPlaces(cityID, 20, discoverArgs)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	playArgs := initializeQueryParams("poi")
	playArgs.Set("categories", "playing")
	playPlaces, err := sygic.GetPlaces(cityID, 20, playArgs)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	eatArgs := initializeQueryParams("poi")
	eatArgs.Set("categories", "eating")
	eatPlaces, err := sygic.GetPlaces(cityID, 20, eatArgs)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	shopArgs := initializeQueryParams("poi")
	shopArgs.Set("categories", "shopping")
	shopPlaces, err := sygic.GetPlaces(cityID, 20, shopArgs)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	sygicCity, err := sygic.GetPlace(cityID)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	city := map[string]interface{}{
		"sygic_id":    sygicCity.Id,
		"name":        sygicCity.Name,
		"image_usage": sygicCity.Main_media.Usage,
		"image":       sygicCity.Main_media.Media[0].Url,
		// TODO: bring over the replace done in TS
		"image_template": sygicCity.Main_media.Media[0].Url_template,
		"description":  sygicCity.Perex,
		"location":     sygicCity.Location,
		"bounding_box": sygicCity.Bounding_box,
	}

	cityData := map[string]interface{}{
		"city": city,

		"see":           FromSygicPlaces(placesToSee),
		"see_locations": location.FromSygicPlaces(placesToSee),

		"discover":           FromSygicPlaces(discoverPlaces),
		"discover_locations": location.FromSygicPlaces(discoverPlaces),

		"play":           FromSygicPlaces(playPlaces),
		"play_locations": location.FromSygicPlaces(playPlaces),

		"eat":           FromSygicPlaces(eatPlaces),
		"eat_locations": location.FromSygicPlaces(eatPlaces),

		"shop":           FromSygicPlaces(shopPlaces),
		"shop_locations": location.FromSygicPlaces(shopPlaces),
	}

	response.Write(w, cityData, http.StatusOK)
	return
}
