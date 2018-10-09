package places

import (
	"net/http"
	"net/url"
	"sync"
	"fmt"
	"sort"

	//"github.com/asqwrd/trotter-api/location"
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
	placeChannel := make(chan triposo.PoiInfo)
	var wg sync.WaitGroup
	var wg2 sync.WaitGroup

	wg.Add(len(popular_countries))

	for _, country := range popular_countries {
		go func(country Place){
			defer wg.Done()
			place, err := triposo.GetPlaceByName(country.Name)
			if err != nil {
				response.WriteErrorResponse(w, err)
				return
			}
			placeChannel <- *place
		}(country)
		
	}

	wg2.Add(len(popular_countries))

	go func() {
		for place := range placeChannel {
			go func(place triposo.PoiInfo){
				defer wg2.Done()
				city, err := triposo.GetDestination(place.Id,"2")
				if err != nil {
					response.WriteErrorResponse(w, err)
					return
				}
				popularCities = append(popularCities, *city...)
			}(place)
		}
	}()
	wg.Wait()
	wg2.Wait()

	sort.Slice(popularCities[:], func(i, j int) bool {
		return popularCities[i].Score > popularCities[j].Score
	})




	responseData := map[string]interface{}{
		"popular_cities":     popularCities,
		"all_countries":      FromSygicPlaces(allCountries),
	}



	response.Write(w, responseData, http.StatusOK)
	fmt.Println("done")
	fmt.Println(len(popular_countries))
	
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
	var wg sync.WaitGroup
	var wg2 sync.WaitGroup
	urlparams := []string{"sightseeing|sight|topattractions","museums|tours|walkingtours|transport|private_tours|celebrations|hoponhopoff|air|architecture|multiday|touristinfo|forts","amusementparks|golf|iceskating|kayaking|sporttickets|sports|surfing|cinema|zoos","beaches|camping|wildlife|fishing|relaxinapark","eatingout|breakfast|coffeeandcake|lunch|dinner","do|shopping","nightlife|comedy|drinks|dancing|pubcrawl|redlight|musicandshows|celebrations|foodexperiences|breweries|showstheatresandmusic"}

	wg.Add(len(urlparams))
	wg2.Add(len(urlparams))
	placeChannel := make(chan triposo.TriposoChannel)
	//cityChannel := make(chan []triposo.Place)
	var placeToSee []triposo.Place
	var discoverPlaces []triposo.Place
	var playPlaces []triposo.Place
	var eatPlaces []triposo.Place
	var nightlifePlaces []triposo.Place
	var shopPlaces []triposo.Place
	var relaxPlaces []triposo.Place

	for i, param := range urlparams {
		go func(param string, i int){
			defer wg.Done()
			place, index, err := triposo.GetPoiFromLocation(cityID,"20",param,i)
			if err != nil {
				response.WriteErrorResponse(w, err)
				return
			}
			res := new(triposo.TriposoChannel)
			res.Places = *place
			res.Index = *index
			placeChannel <- *res
		}(param, i)
		
	}

	go func() {
		for res := range placeChannel {
			switch {
			case res.Index == 1:
				discoverPlaces = res.Places
			case res.Index == 2:
				playPlaces = res.Places
			case res.Index == 3:
				eatPlaces = res.Places
			case res.Index == 4:
				nightlifePlaces = res.Places
			case res.Index == 5:
				shopPlaces = res.Places
			case res.Index == 6:
				relaxPlaces = res.Places
			default:
				placeToSee = res.Places
			}
			
		}
	}()
	var city []triposo.Place
	go func(city []triposo.Place){
		defer wg2.Done()
		res, err := triposo.GetCity(cityID)
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		city = *res
	}(city)

	wg.Wait()
	wg2.Wait()


	cityData := map[string]interface{}{
		"city": city,

		"see":           placeToSee,
		//"see_locations": location.FromSygicPlaces(placesToSee),

		"discover":           discoverPlaces,
	//	"discover_locations": location.FromSygicPlaces(discoverPlaces),

		"play":           playPlaces,
		//"play_locations": location.FromSygicPlaces(playPlaces),

		"eat":           eatPlaces,
		//"eat_locations": location.FromSygicPlaces(eatPlaces),

		"shop":           shopPlaces,
		//"shop_locations": location.FromSygicPlaces(shopPlaces),
	}

	response.Write(w, cityData, http.StatusOK)
	return
}
