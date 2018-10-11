package places

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"

	//"sync"

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
	popularCities := []triposo.Place{}
	placeChannel := make(chan triposo.PoiInfo)
	allCountryChannel := make(chan []sygic.Place)
	citiesChannel := make(chan []triposo.Place, 5)
	var popularCountries []Place
	var allCountries []Place

	go func() {
		allCountriesArgs := initializeQueryParams("country")
		res, err := sygic.GetPlaces(parentID, 60, allCountriesArgs)
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		allCountryChannel <- res
	}()

	select {
	case res1 := <-allCountryChannel:
		allCountries = FromSygicPlaces(res1)
		popularCountries = allCountries[:5]
	}

	for _, country := range popularCountries {
		go func(country Place) {
			place, err := triposo.GetPlaceByName(country.Name)
			if err != nil {
				response.WriteErrorResponse(w, err)
				return
			}
			placeChannel <- *place
		}(country)

	}

	go func() {
		for place := range placeChannel {
			go func(place triposo.PoiInfo) {
				city, err := triposo.GetDestination(place.Id, "2")
				if err != nil {
					response.WriteErrorResponse(w, err)
					return
				}
				citiesChannel <- *city
			}(place)
		}
	}()

	for i := 0; i < 5; i++ {
		select {
		case city := <-citiesChannel:
			popularCities = append(popularCities, city...)
		}
	}

	sort.Slice(popularCities[:], func(i, j int) bool {
		return popularCities[i].Score > popularCities[j].Score
	})

	responseData := map[string]interface{}{
		"popular_cities": FromTriposoPlaces(popularCities),
		"all_countries":  allCountries,
	}

	response.Write(w, responseData, http.StatusOK)
	fmt.Println("done")

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
	urlparams := []string{"sightseeing|sight|topattractions", "museums|tours|walkingtours|transport|private_tours|celebrations|hoponhopoff|air|architecture|multiday|touristinfo|forts", "amusementparks|golf|iceskating|kayaking|sporttickets|sports|surfing|cinema|zoos", "beaches|camping|wildlife|fishing|relaxinapark", "eatingout|breakfast|coffeeandcake|lunch|dinner", "do|shopping", "nightlife|comedy|drinks|dancing|pubcrawl|redlight|musicandshows|celebrations|foodexperiences|breweries|showstheatresandmusic"}

	placeChannel := make(chan triposo.TriposoChannel)
	cityChannel := make(chan []triposo.Place)
	var city *triposo.InternalPlace

	var placeToSee []triposo.InternalPlace
	var discoverPlaces []triposo.InternalPlace
	var playPlaces []triposo.InternalPlace
	var eatPlaces []triposo.InternalPlace
	var nightlifePlaces []triposo.InternalPlace
	var shopPlaces []triposo.InternalPlace
	var relaxPlaces []triposo.InternalPlace

	seeChannel := make(chan []triposo.Place)
	eatChannel := make(chan []triposo.Place)
	discoverChannel := make(chan []triposo.Place)
	playChannel := make(chan []triposo.Place)
	nightlifeChannel := make(chan []triposo.Place)
	shopChannel := make(chan []triposo.Place)
	relaxChannel := make(chan []triposo.Place)

	for i, param := range urlparams {
		go func(param string, i int) {
			place, err := triposo.GetPoiFromLocation(cityID, "20", param, i)
			if err != nil {
				response.WriteErrorResponse(w, err)
				return
			}
			res := new(triposo.TriposoChannel)
			res.Places = *place
			res.Index = i
			placeChannel <- *res
		}(param, i)

	}

	go func() {
		for res := range placeChannel {
			switch {
			case res.Index == 0:
				seeChannel <- res.Places
			case res.Index == 1:
				discoverChannel <- res.Places
			case res.Index == 2:
				playChannel <- res.Places
			case res.Index == 3:
				eatChannel <- res.Places
			case res.Index == 4:
				nightlifeChannel <- res.Places
			case res.Index == 5:
				shopChannel <- res.Places
			case res.Index == 6:
				relaxChannel <- res.Places
			}
		}

	}()

	go func() {
		city, err := triposo.GetCity(cityID)
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		cityChannel <- *city

	}()

	for i := 0; i < 8; i++ {
		select {
		case see := <-seeChannel:
			placeToSee = FromTriposoPlaces(see)
		case eat := <-eatChannel:
			eatPlaces = FromTriposoPlaces(eat)
		case discover := <-discoverChannel:
			discoverPlaces = FromTriposoPlaces(discover)
		case shop := <-shopChannel:
			shopPlaces = FromTriposoPlaces(shop)
		case relax := <-relaxChannel:
			relaxPlaces = FromTriposoPlaces(relax)
		case play := <-playChannel:
			playPlaces = FromTriposoPlaces(play)
		case nightlife := <-nightlifeChannel:
			nightlifePlaces = FromTriposoPlaces(nightlife)
		case cityRes := <-cityChannel:
			city = FromTriposoPlace(&cityRes[0])
		}
	}

	cityData := map[string]interface{}{
		"city": city,

		"see":           &placeToSee,
		"see_locations": location.FromTriposoPlaces(placeToSee),

		"discover":           &discoverPlaces,
		"discover_locations": location.FromTriposoPlaces(discoverPlaces),

		"play":           &playPlaces,
		"play_locations": location.FromTriposoPlaces(playPlaces),

		"eat":           &eatPlaces,
		"eat_locations": location.FromTriposoPlaces(eatPlaces),

		"shop":           &shopPlaces,
		"shop_locations": location.FromTriposoPlaces(shopPlaces),

		"nightlife":      &nightlifePlaces,
		"nightlife_locations": location.FromTriposoPlaces(nightlifePlaces),

		"relax":           &relaxPlaces,
		"relax_locations": location.FromTriposoPlaces(relaxPlaces),
	}

	response.Write(w, cityData, http.StatusOK)
	return
}
