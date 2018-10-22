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
		"popular_cities": FromTriposoPlaces(popularCities, "city"),
		"all_countries":  allCountries,
	}

	response.Write(w, responseData, http.StatusOK)
	fmt.Println("done")

	return
}

//Get City

func GetCity(w http.ResponseWriter, r *http.Request) {
	cityID := mux.Vars(r)["cityID"]
	urlparams := []string{"sightseeing|sight|topattractions", "museums|tours|walkingtours|transport|private_tours|celebrations|hoponhopoff|air|architecture|multiday|touristinfo|forts", "amusementparks|golf|iceskating|kayaking|sporttickets|sports|surfing|cinema|zoos", "beaches|camping|wildlife|fishing|relaxinapark", "eatingout|breakfast|coffeeandcake|lunch|dinner", "do|shopping", "nightlife|comedy|drinks|dancing|pubcrawl|redlight|musicandshows|celebrations|foodexperiences|breweries|showstheatresandmusic"}

	placeChannel := make(chan triposo.TriposoChannel)
	cityChannel := make(chan triposo.InternalPlace)
	colorChannel := make(chan Colors)
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
	var cityColor string

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
				return
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

		cityParam := *city
		cityRes := FromTriposoPlace(&cityParam[0], "city")

		go func(image string) {
			colors, err := GetColor(image)
			if err != nil {
				errorChannel <- err
				return
			}
			colorChannel <- *colors
		}(cityRes.Image)

		cityChannel <- *cityRes

	}()

	go func() {
		time.Sleep(10 * time.Second)
		timeoutChannel <- true
	}()

	for i := 0; i < 9; i++ {
		select {
		case see := <-seeChannel:
			placeToSee = FromTriposoPlaces(see, "poi")
		case eat := <-eatChannel:
			eatPlaces = FromTriposoPlaces(eat, "poi")
		case discover := <-discoverChannel:
			discoverPlaces = FromTriposoPlaces(discover, "poi")
		case shop := <-shopChannel:
			shopPlaces = FromTriposoPlaces(shop, "poi")
		case relax := <-relaxChannel:
			relaxPlaces = FromTriposoPlaces(relax, "poi")
		case play := <-playChannel:
			playPlaces = FromTriposoPlaces(play, "poi")
		case nightlife := <-nightlifeChannel:
			nightlifePlaces = FromTriposoPlaces(nightlife, "poi")
		case cityRes := <-cityChannel:
			city = &cityRes
		case colorRes := <-colorChannel:
			if len(colorRes.Vibrant) > 0 {
				cityColor = colorRes.Vibrant
			} else if len(colorRes.Muted) > 0 {
				cityColor = colorRes.Muted
			} else if len(colorRes.LightVibrant) > 0 {
				cityColor = colorRes.LightVibrant
			} else if len(colorRes.LightMuted) > 0 {
				cityColor = colorRes.LightMuted
			} else if len(colorRes.DarkVibrant) > 0 {
				cityColor = colorRes.DarkVibrant
			} else if len(colorRes.DarkMuted) > 0 {
				cityColor = colorRes.DarkMuted
			}
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
		"city":  city,
		"color": cityColor,

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

		"nightlife":           &nightlifePlaces,
		"nightlife_locations": location.FromTriposoPlaces(nightlifePlaces),

		"relax":           &relaxPlaces,
		"relax_locations": location.FromTriposoPlaces(relaxPlaces),
	}

	response.Write(w, cityData, http.StatusOK)
	return
}

//Get Home

func GetHome(w http.ResponseWriter, r *http.Request) {
	typeparams := []string{"island", "city", "country"}

	placeChannel := make(chan PlaceChannel)

	var islands []triposo.InternalPlace
	var cities []triposo.InternalPlace
	var countries []Place

	islandChannel := make(chan []triposo.Place)
	cityChannel := make(chan []triposo.Place)
	countryChannel := make(chan []sygic.Place)

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
			} else {
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
				return
			}
			switch {
			case res.Index == 0:
				islandChannel <- res.Places.([]triposo.Place)
			case res.Index == 1:
				cityChannel <- res.Places.([]triposo.Place)
			case res.Index == 2:
				countryChannel <- res.Places.([]sygic.Place)
			}
		}

	}()

	go func() {
		time.Sleep(10 * time.Second)
		timeoutChannel <- true
	}()

	for i := 0; i < 3; i++ {
		select {
		case res := <-islandChannel:
			islands = FromTriposoPlaces(res, "island")
		case res := <-countryChannel:
			countries = FromSygicPlaces(res)
		case res := <-cityChannel:
			cities = FromTriposoPlaces(res, "city")
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
		"popular_cities": cities,

		"popular_islands": islands,

		"popular_countries": countries,
	}

	response.Write(w, homeData, http.StatusOK)
	return
}

//POI

func GetPoi(w http.ResponseWriter, r *http.Request) {
	poiID := mux.Vars(r)["poiID"]
	poiChannel := make(chan triposo.InternalPlace)
	colorChannel := make(chan Colors)
	errorChannel := make(chan error)
	timeoutChannel := make(chan bool)
	var poiColor string
	var poi *triposo.InternalPlace

	go func() {
		poi, err := triposo.GetPoi(poiID)
		if err != nil {
			errorChannel <- err
			return
		}
		poiParam := *poi
		poiRes := FromTriposoPlace(&poiParam[0], "poi")

		go func(image string) {
			colors, err := GetColor(image)
			if err != nil {
				errorChannel <- err
				return
			}
			colorChannel <- *colors
		}(poiRes.Image)
		poiChannel <- *poiRes

	}()

	go func() {
		time.Sleep(10 * time.Second)
		timeoutChannel <- true
	}()

	for i := 0; i < 2; i++ {
		select {
		case poiRes := <-poiChannel:
			poi = &poiRes
		case color := <-colorChannel:
			if len(color.Vibrant) > 0 {
				poiColor = color.Vibrant
			} else if len(color.Muted) > 0 {
				poiColor = color.Muted
			} else if len(color.LightVibrant) > 0 {
				poiColor = color.LightVibrant
			} else if len(color.LightMuted) > 0 {
				poiColor = color.LightMuted
			} else if len(color.DarkVibrant) > 0 {
				poiColor = color.DarkVibrant
			} else if len(color.DarkMuted) > 0 {
				poiColor = color.DarkMuted
			}
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

	poiData := map[string]interface{}{
		"poi":   poi,
		"color": poiColor,
	}

	response.Write(w, poiData, http.StatusOK)
	return
}
