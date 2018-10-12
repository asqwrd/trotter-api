package places

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"
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
	timeoutChannel := make(chan bool)
	errorChannel := make(chan error)
	var popularCountries []Place
	var allCountries []Place

	go func() {
		allCountriesArgs := initializeQueryParams("country")
		res, err := sygic.GetPlaces(parentID, 60, allCountriesArgs)
		if err != nil {
			errorChannel <- err
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
				errorChannel <- err
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
					errorChannel <- err
					return
				}
				citiesChannel <- *city
			}(place)
		}
	}()

	go func() {
    time.Sleep(10 * time.Second)
    timeoutChannel <- true
	}()

	for i := 0; i < 5; i++ {
		select {
		case city := <-citiesChannel:
			popularCities = append(popularCities, city...)
		case err := <-errorChannel:
			response.WriteErrorResponse(w, err)
			return
		case timeout := <-timeoutChannel:
			if timeout == true {
				response.WriteErrorResponse(w, fmt.Errorf("api timeout"))
				return
			}
			
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

//Get Country

func GetCountry(w http.ResponseWriter, r *http.Request) {
	color := GetColor("http://api-images-www.triposo.com/20180627/gAAAAABbwMi0wMG8hb-t08NKGG0jxwI4IJf3gaxk6sZMLGsKDS1XIo_QYhREQWv4qdLFtasSKn8YGWdJyJm_vxWFopD_3xtS80Q9AFlsjtAGvmopEjVxbLeKIjrJ6f9mTQfPRkzPxOXak9G2JBUjYOY9WrNNKrvHwJ70CwMNH9kOTqv7C0Dr5DbowBaF65IC-T_qZyXYqZJ7wzKRIlZHVFmzaQvyHmOyHrFYO4ocYIrh-deK2_3ZZFZupBYhy_gl4cFwZL4F78e45-4kU3W8q2xWw3SEACDKuBfmWH-PR1IkQMaElSX40u4=")

	data := map[string]interface{}{
		"color": color,
	}
	
	
	response.Write(w, data, http.StatusOK)
	return
}

//Get City

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
	errorChannel := make(chan error)
	timeoutChannel := make(chan bool)

	for i, param := range urlparams {
		go func(param string, i int) {
			place, err := triposo.GetPoiFromLocation(cityID, "20", param, i)
			res := new(triposo.TriposoChannel)
			res.Places = *place
			res.Index = i
			res.Error = err
			placeChannel <- *res
		}(param, i)

	}

	go func() {
		for res := range placeChannel {
			if res.Error != nil {
				errorChannel <- res.Error
				return;
			}
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
			errorChannel <- err
			return
		}
		cityChannel <- *city

	}()

	go func() {
    time.Sleep(10 * time.Second)
    timeoutChannel <- true
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
			city.Colors = GetColor(city.Image)
		case err := <-errorChannel:
			response.WriteErrorResponse(w, err)
			return
		case timeout := <-timeoutChannel:
			if timeout == true {
				response.WriteErrorResponse(w, fmt.Errorf("api timeout"))
				return
			}
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

//Get Home

func GetHome(w http.ResponseWriter, r *http.Request){
	typeparams := []string{"island","city","country","national_park"}

	placeChannel := make(chan PlaceChannel)

	var islands []triposo.InternalPlace
	var cities []triposo.InternalPlace
	var countries []Place
	var nationalParks []triposo.InternalPlace

	islandChannel := make(chan []triposo.Place)
	cityChannel := make(chan []triposo.Place)
	countryChannel := make(chan []sygic.Place)
	nationalParkChannel := make(chan []triposo.Place)
	
	errorChannel := make(chan error)
	timeoutChannel := make(chan bool)

	for i, typeParam := range typeparams {
		go func(typeParam string, i int) {
			if typeParam == "country" {
				place, err := sygic.GetCountry("20")
				res := new(PlaceChannel)
				res.Places = *place
				res.Index = i
				res.Error = err
				placeChannel <- *res
			}else{
				place, err := triposo.GetLocationType(typeParam, "20")
				res := new(PlaceChannel)
				res.Places = *place
				res.Index = i
				res.Error = err
				placeChannel <- *res
			}
			
			
		}(typeParam, i)

	}

	go func() {
		for res := range placeChannel {
			if res.Error != nil {
				errorChannel <- res.Error
				return;
			}
			switch {
			case res.Index == 0:
				islandChannel <- res.Places.([]triposo.Place)
			case res.Index == 1:
				cityChannel <- res.Places.([]triposo.Place)
			case res.Index == 2:
				countryChannel <- res.Places.([]sygic.Place)
			case res.Index == 3:
				nationalParkChannel <- res.Places.([]triposo.Place)
			}
		}

	}()

	go func() {
    time.Sleep(10 * time.Second)
    timeoutChannel <- true
	}()

	for i := 0; i < 4; i++ {
		select {
		case res := <-islandChannel:
			islands = FromTriposoPlaces(res)
		case res := <-countryChannel:
			countries = FromSygicPlaces(res)
		case res := <-nationalParkChannel:
			nationalParks = FromTriposoPlaces(res)
		case res := <-cityChannel:
			cities = FromTriposoPlaces(res)
		case err := <-errorChannel:
			response.WriteErrorResponse(w, err)
			return
		case timeout := <-timeoutChannel:
			if timeout == true {
				response.WriteErrorResponse(w, fmt.Errorf("api timeout"))
				return
			}
		}
	}

	homeData := map[string]interface{}{
		"popular_cities": 						cities,

		"popular_islands":           	islands,

		"popular_national_parks":    	nationalParks,

		"popular_countries":          countries,

	}

	response.Write(w, homeData, http.StatusOK)
	return
}


//POI

func GetPoi(w http.ResponseWriter, r *http.Request) {
	poiID := mux.Vars(r)["poiID"]
	poiChannel := make(chan []triposo.Place)
	errorChannel := make(chan error)
	timeoutChannel := make(chan bool)
	var poi *triposo.InternalPlace


	go func() {
		poi, err := triposo.GetPoi(poiID)
		if err != nil {
			errorChannel <- err
			return
		}
		poiChannel <- *poi

	}()

	go func() {
    time.Sleep(10 * time.Second)
    timeoutChannel <- true
	}()

	for i := 0; i < 1; i++ {
		select {
		case poiRes := <-poiChannel:
			poi = FromTriposoPlace(&poiRes[0])
			poi.Colors = GetColor(poi.Image)
		case err := <-errorChannel:
			response.WriteErrorResponse(w, err)
			return
		case timeout := <-timeoutChannel:
			if timeout == true {
				response.WriteErrorResponse(w, fmt.Errorf("api timeout"))
				return
			}
		}
	}

	response.Write(w, poi, http.StatusOK)
	return
}
