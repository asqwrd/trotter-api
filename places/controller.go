package places

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"strconv"


	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/location"
	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/sygic"
	"github.com/asqwrd/trotter-api/triposo"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"googlemaps.github.io/maps"
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
				city, err := triposo.GetDestination(place.ID, "2")
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
	urlparams := []string{"sightseeing|sight|topattractions",
		"museums|tours|walkingtours|transport|private_tours|celebrations|hoponhopoff|air|architecture|multiday|touristinfo|forts",
		"amusementparks|golf|iceskating|kayaking|sporttickets|sports|surfing|cinema|zoos",
		"beaches|camping|wildlife|fishing|relaxinapark",
		"eatingout|breakfast|coffeeandcake|lunch|dinner",
		"do|shopping",
		"nightlife|comedy|drinks|dancing|pubcrawl|redlight|musicandshows|celebrations|foodexperiences|breweries|showstheatresandmusic"}

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
			case res.Index == 4:
				eatChannel <- res.Places
			case res.Index == 6:
				nightlifeChannel <- res.Places
			case res.Index == 5:
				shopChannel <- res.Places
			case res.Index == 3:
				relaxChannel <- res.Places
			}
		}

	}()

	go func() {
		city, err := triposo.GetLocation(cityID)
		if err != nil {
			//fmt.Println("here")
			errorChannel <- err
			return
		}

		cityParam := *city
		cityRes := FromTriposoPlace(cityParam[0], "city")
		country, err := triposo.GetLocation(cityRes.CountryID);
		if err != nil {
			errorChannel <- err
			return
		}

		countryParam := *country
		cityRes.CountryName = countryParam[0].Name

		go func(image string) {
			if len(image) == 0 {
				var colors Colors
				colors.Vibrant = "#c27949"
				colorChannel <- colors
			} else {
				colors, err := GetColor(image)
				if err != nil {
					errorChannel <- err
					return
				}
				colorChannel <- *colors
			}
		}(cityRes.Image)

		cityChannel <- cityRes

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

//GetHome
func GetHome(w http.ResponseWriter, r *http.Request) {
	typeparams := []string{"island", "city"}
	fmt.Println("Got Home")

	placeChannel := make(chan PlaceChannel)

	var islands []triposo.InternalPlace
	var cities []triposo.InternalPlace
	//var national_parks []triposo.InternalPlace
	//var countries []triposo.InternalPlace

	islandChannel := make(chan []triposo.Place)
	cityChannel := make(chan []triposo.Place)
	//parkChannel := make(chan []triposo.Place)
	//countryChannel := make(chan []triposo.Place)

	errorChannel := make(chan error)
	timeoutChannel := make(chan bool)

	for i, typeParam := range typeparams {
		go func(typeParam string, i int) {
			place, err := triposo.GetLocationType(typeParam, "20")
			res := new(PlaceChannel)
			res.Places = *place
			res.Index = i
			res.Error = err
			placeChannel <- *res
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
			}
		}
	}()

	go func() {
		time.Sleep(10 * time.Second)
		timeoutChannel <- true
	}()

	for i := 0; i < 2; i++ {
		select {
		case res := <-islandChannel:
			islands = FromTriposoPlaces(res, "island")
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

	cityParentChannel := make(chan PlaceChannel)
	go func(){
		for i := 0; i < len(cities); i++ {
			go func(index int) {
				countryID := cities[index].CountryID
				if countryID == "United_States" {
					countryID = cities[index].ParentID
				}
				country, err := triposo.GetLocation(countryID)
				res := new(PlaceChannel)
				res.Places = *country
				res.Index = index
				res.Error = err
				cityParentChannel <- *res
			}(i)
		}
	}()

	islandParentChannel := make(chan PlaceChannel)
	go func(){
		for i := 0; i < len(islands); i++ {
			go func(index int) {
				country_id := islands[index].CountryID
				if country_id == "United_States" {
					country_id = islands[index].ParentID
				}
				country, err := triposo.GetLocation(country_id)
				res := new(PlaceChannel)
				res.Places = *country
				res.Index = index
				res.Error = err
				islandParentChannel <- *res
			}(i)
		}
	}()


	for i := 0; i < len(cities); i++ {
		select{
		case res := <- cityParentChannel:
			if res.Error != nil {
				response.WriteErrorResponse(w, res.Error)
				return
			}
			cities[res.Index].CountryName = res.Places.([]triposo.Place)[0].Name
			if cities[res.Index].CountryID == "United_States" {
				cities[res.Index].CountryName = "United States"
			}
			cities[res.Index].ParentName = res.Places.([]triposo.Place)[0].Name 
		}
	}

	for i := 0; i < len(islands); i++ {
		select{
		case res := <- islandParentChannel:
			if res.Error != nil {
				response.WriteErrorResponse(w, res.Error)
				return
			}
			islands[res.Index].CountryName = res.Places.([]triposo.Place)[0].Name
			if islands[res.Index].CountryID == "United_States" {
				islands[res.Index].CountryName = "United States"
			}
			islands[res.Index].ParentName = res.Places.([]triposo.Place)[0].Name 
		}
	}

	homeData := map[string]interface{}{
		"popular_cities": cities,

		"popular_islands": islands,
	}

	response.Write(w, homeData, http.StatusOK)
	fmt.Println("home done")
	return
}

//GetPoi func
func GetPoi(w http.ResponseWriter, r *http.Request) {
	poiID := mux.Vars(r)["poiID"]
	googlePlace := r.URL.Query().Get("googlePlace")
	locationId := r.URL.Query().Get("locationId")
	poiChannel := make(chan triposo.InternalPlace)
	colorChannel := make(chan Colors)
	errorChannel := make(chan error)
	timeoutChannel := make(chan bool)
	var poiColor string
	var poi *triposo.InternalPlace
	ctx := context.Background()

	if googlePlace == "true" {
		go func() {
			googleClient, err := InitGoogle()
			if err != nil  {
				errorChannel <- err
				return
			}
			r := &maps.PlaceDetailsRequest{
				PlaceID:      poiID,
			}
			place,err := googleClient.PlaceDetails(ctx,r)
			if err != nil {
				errorChannel <- err
				return
			}
			photo := "https://maps.googleapis.com/maps/api/place/photo?maxwidth=1280&photoreference="+place.Photos[0].PhotoReference+"&key="+GoogleApi
			go func(image string) {
				if len(image) == 0 {
					var colors Colors
					colors.Vibrant = "#c27949"
					colorChannel <- colors
				} else {
					colors, err := GetColor(image)
					if err != nil {
						errorChannel <- err
						return
					}
					colorChannel <- *colors
				}
			}(photo)
			//poiChannel <- poiRes
			poiChannel <- FromGooglePlace(place, "poi")
			
		}()

		
	} else {
		go func() {
			poi, err := triposo.GetPoi(poiID)
			if err != nil {
				errorChannel <- err
				return
			}
			poiParam := *poi
			poiRes := FromTriposoPlace(poiParam[0], "poi")
	
			go func(image string) {
				if len(image) == 0 {
					var colors Colors
					colors.Vibrant = "#c27949"
					colorChannel <- colors
				} else {
					colors, err := GetColor(image)
					if err != nil {
						errorChannel <- err
						return
					}
					colorChannel <- *colors
				}
			}(poiRes.Image)
			poiChannel <- poiRes
	
		}()
	}
	

	go func() {
		time.Sleep(30 * time.Second)
		timeoutChannel <- true
	}()

	for i := 0; i < 2; i++ {
		select {
		case poiRes := <-poiChannel:
			poi = &poiRes
			if googlePlace == "true" {
				poi.LocationID = locationId
			}
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

//Get Park

func GetPark(w http.ResponseWriter, r *http.Request) {
	parkID := mux.Vars(r)["parkID"]

	parkChannel := make(chan triposo.InternalPlace)
	colorChannel := make(chan Colors)
	var park *triposo.InternalPlace

	var pois []triposo.InternalPlace

	poiChannel := make(chan []triposo.Place)
	errorChannel := make(chan error)
	timeoutChannel := make(chan bool)
	var parkColor string

	go func() {
		place, err := triposo.GetPoiFromLocation(parkID, "20", "", 0)
		if err != nil {
			errorChannel <- err
			return
		}
		poiChannel <- *place
	}()

	go func() {
		parkData, err := triposo.GetLocation(parkID)
		if err != nil {
			errorChannel <- err
			return
		}

		parkParam := *parkData
		parkRes := FromTriposoPlace(parkParam[0], "national_park")

		go func(image string) {
			if len(image) == 0 {
				var colors Colors
				colors.Vibrant = "#c27949"
				colorChannel <- colors
			} else {
				colors, err := GetColor(image)
				if err != nil {
					errorChannel <- err
					return
				}
				colorChannel <- *colors
			}
		}(parkRes.Image)

		parkChannel <- parkRes

	}()

	go func() {
		time.Sleep(10 * time.Second)
		timeoutChannel <- true
	}()

	for i := 0; i < 3; i++ {
		select {
		case poi := <-poiChannel:
			pois = FromTriposoPlaces(poi, "poi")
		case parkRes := <-parkChannel:
			park = &parkRes
		case colorRes := <-colorChannel:
			if len(colorRes.Vibrant) > 0 {
				parkColor = colorRes.Vibrant
			} else if len(colorRes.Muted) > 0 {
				parkColor = colorRes.Muted
			} else if len(colorRes.LightVibrant) > 0 {
				parkColor = colorRes.LightVibrant
			} else if len(colorRes.LightMuted) > 0 {
				parkColor = colorRes.LightMuted
			} else if len(colorRes.DarkVibrant) > 0 {
				parkColor = colorRes.DarkVibrant
			} else if len(colorRes.DarkMuted) > 0 {
				parkColor = colorRes.DarkMuted
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

	parkData := map[string]interface{}{
		"park":  park,
		"color": parkColor,

		"pois":          &pois,
		"poi_locations": location.FromTriposoPlaces(pois),
	}

	response.Write(w, parkData, http.StatusOK)
	return
}

// Search function
func Search(w http.ResponseWriter, r *http.Request) {
	query := mux.Vars(r)["query"]
	latq :=r.URL.Query().Get("lat")
	lngq :=r.URL.Query().Get("lng")
	fmt.Println(latq)

	lat,_ := strconv.ParseFloat(latq,64)
	lng, _ := strconv.ParseFloat(lngq,64)

	//id := r.URL.Query().Get("id")
	//poi := r.URL.Query().Get("poi")
	errorChannel := make(chan error)
	timeoutChannel := make(chan bool)
	addQuery := make(chan bool)
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	if (len(latq) == 0 && len(lngq) == 0) {
		typeparams := []string{"island", "city", "city_state", "national_park", "region"}
		placeChannel := make(chan PlaceChannel)

		var triposoResults []triposo.InternalPlace

		islandChannel := make(chan []triposo.Place)
		cityChannel := make(chan []triposo.Place)
		cityStateChannel := make(chan []triposo.Place)
		parkChannel := make(chan []triposo.Place)
		regionChannel := make(chan []triposo.Place)
		//poiChannel := make(chan []triposo.Place)
		//countryChannel := make(chan []triposo.Place)

		go func() {
			search, err := client.Collection("searches").Doc(strings.ToUpper(query)).Get(ctx)
			if err != nil {
				addQuery <- true
				return
			}

			searchData := search.Data()
			count := searchData["count"].(int64) + 1
			_, errSearch := client.Collection("searches").Doc(strings.ToUpper(query)).Set(ctx, map[string]interface{}{
				"count": count,
				"value": query,
			})
			if errSearch != nil {
				// Handle any errors in an appropriate way, such as returning them.
				response.WriteErrorResponse(w, errSearch)
			}
			addQuery <- false

		}()

		for i, typeParam := range typeparams {
			go func(typeParam string, i int) {
				place, err := triposo.Search(query, typeParam, "")
				res := new(PlaceChannel)
				res.Places = *place
				res.Index = i
				res.Error = err
				placeChannel <- *res
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
					cityStateChannel <- res.Places.([]triposo.Place)
				case res.Index == 3:
					parkChannel <- res.Places.([]triposo.Place)
				case res.Index == 4:
					regionChannel <- res.Places.([]triposo.Place)
				}
			}

		}()

		go func() {
			time.Sleep(10 * time.Second)
			timeoutChannel <- true
		}()

		for i := 0; i < 5; i++ {
			select {
			case res := <-islandChannel:
				triposoResults = append(triposoResults, FromTriposoPlaces(res, "island")...)
			case res := <-cityChannel:
				triposoResults = append(triposoResults, FromTriposoPlaces(res, "city")...)
			case res := <-cityStateChannel:
				triposoResults = append(triposoResults, FromTriposoPlaces(res, "city_state")...)
			case res := <-parkChannel:
				triposoResults = append(triposoResults, FromTriposoPlaces(res, "national_park")...)
			case res := <-regionChannel:
				triposoResults = append(triposoResults, FromTriposoPlaces(res, "region")...)
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

		resChannel := make(chan InternalPlaceChannel)

		for index := 0; index < len(triposoResults); index++ {
			go func(index int) {
				
				if triposoResults[index].CountryID == "United_States" {
					parent, err := triposo.GetLocation(triposoResults[index].ParentID)
					res := new(InternalPlaceChannel)
					parentParam := *parent
					res.Place = FromTriposoPlace(parentParam[0], "")
					res.Index = index
					res.Error = err
					triposoResults[index].CountryName = "United States"

					resChannel <- *res
				} else {
					parent, err := triposo.GetLocation(triposoResults[index].CountryID)
					res := new(InternalPlaceChannel)
					parentParam := *parent
					res.Place = FromTriposoPlace(parentParam[0], "")
					res.Index = index
					res.Error = err
					resChannel <- *res
				}

			}(index)
		}

		for i := 0; i < len(triposoResults); i++ {
			select {
			case res := <-resChannel:
				if res.Error != nil {
					response.WriteErrorResponse(w, res.Error)
					return
				}
				triposoResults[res.Index].ParentName = res.Place.Name
				if len(triposoResults[res.Index].CountryName) == 0 {
					triposoResults[res.Index].CountryName = res.Place.Name
				}
			}
		}

		for i := 0; i < 1; i++ {
			select {
			case res := <-addQuery:
				if res == true && (len(triposoResults) > 0) {
					_, err := client.Collection("searches").Doc(strings.ToUpper(query)).Set(ctx, map[string]interface{}{
						"count": 1,
						"value": query,
					})
					if err != nil {
						// Handle any errors in an appropriate way, such as returning them.
						response.WriteErrorResponse(w, err)
					}
				}
			}
		}
		sort.Slice(triposoResults, 
			func(i, j int) bool { 
				return triposoResults[i].Trigram > triposoResults[j].Trigram 
			})

		searchData := map[string]interface{}{
			"results": triposoResults,
		}

		response.Write(w, searchData, http.StatusOK)
		return
	} else {
		var triposoResults []triposo.InternalPlace
		//poiChannel := make(chan []triposo.Place)

		go func() {
			search, err := client.Collection("searches_poi").Doc(strings.ToUpper(query)).Get(ctx)
			if err != nil {
				addQuery <- true
				return
			}

			searchData := search.Data()
			count := searchData["count"].(int64) + 1
			_, errSearch := client.Collection("searches_poi").Doc(strings.ToUpper(query)).Set(ctx, map[string]interface{}{
				"count": count,
				"value": query,
			})
			if errSearch != nil {
				// Handle any errors in an appropriate way, such as returning them.
				response.WriteErrorResponse(w, errSearch)
			}
			addQuery <- false

		}()

		googlePlaceChannel := make(chan triposo.InternalPlace)
		googleClient, err := InitGoogle()
		if err != nil  {
			errorChannel <- err
			return
		}
		latlng := &maps.LatLng{Lat: lat, Lng: lng}
		r := &maps.PlaceAutocompleteRequest{
			Input: query,
			Location: latlng,
			Radius : 50000,
		}
		places,err := googleClient.PlaceAutocomplete(ctx,r)
				if err != nil {
					errorChannel <- err
					return
				}
		for i:=0; i < len(places.Predictions); i++ {
			go func(placeID string) {
				r := &maps.PlaceDetailsRequest{
					PlaceID:      placeID,
				}
				place,err := googleClient.PlaceDetails(ctx,r)
				googlePlaceChannel <- FromGooglePlace(place,"poi")
				if err != nil {
					errorChannel <- err
					return
				}
			}(places.Predictions[i].PlaceID)
		}

		go func() {
			time.Sleep(10 * time.Second)
			timeoutChannel <- true
		}()

		for i := 0; i <  len(places.Predictions); i++ {
			select {
			case res := <-googlePlaceChannel:
				triposoResults = append(triposoResults, res)
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

		for i := 0; i < 1; i++ {
			select {
			case res := <-addQuery:
				if res == true && (len(triposoResults) > 0) {
					_, err := client.Collection("searches_poi").Doc(strings.ToUpper(query)).Set(ctx, map[string]interface{}{
						"count": 1,
						"value": query,
					})
					if err != nil {
						// Handle any errors in an appropriate way, such as returning them.
						response.WriteErrorResponse(w, err)
					}
				}
			}
		}

		response.Write(w, map[string]interface{}{
			"results": triposoResults,
		}, http.StatusOK)
		return
	}
}

// SearchGoogle function
func SearchGoogle(w http.ResponseWriter, r *http.Request) {
	query := mux.Vars(r)["query"]
	latq :=r.URL.Query().Get("lat")
	lngq :=r.URL.Query().Get("lng")

	lat,errlat := strconv.ParseFloat(latq,64)
	lng, errlng := strconv.ParseFloat(lngq,64)
	if errlng != nil {
		response.WriteErrorResponse(w, errlng)
		return
	}
	if errlat != nil {
		response.WriteErrorResponse(w, errlat)
		return
	}

	addQuery := make(chan bool)
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()
		var triposoResults []triposo.InternalPlace

		go func() {
			search, err := client.Collection("searches_poi").Doc(strings.ToUpper(query)).Get(ctx)
			if err != nil {
				addQuery <- true
				return
			}

			searchData := search.Data()
			count := searchData["count"].(int64) + 1
			_, errSearch := client.Collection("searches_poi").Doc(strings.ToUpper(query)).Set(ctx, map[string]interface{}{
				"count": count,
				"value": query,
			})
			if errSearch != nil {
				// Handle any errors in an appropriate way, such as returning them.
				response.WriteErrorResponse(w, errSearch)
			}
			addQuery <- false

		}()

		googleClient, err := InitGoogle()
		if err != nil  {
			response.WriteErrorResponse(w, err)
		}
		latlng := &maps.LatLng{Lat: lat, Lng: lng}
		p := &maps.TextSearchRequest {
			Query: query,
			Location : latlng,
			Radius  : 50000,
		}
		places,err := googleClient.TextSearch(ctx,p)
		if err != nil {
			response.WriteErrorResponse(w, err)
		}
		for i:=0; i < len(places.Results); i++ {
			triposoResults = append(triposoResults, FromGooglePlaceSearch(places.Results[i],"poi"))		
		}


		for i := 0; i < 1; i++ {
			select {
			case res := <-addQuery:
				if res == true && (len(triposoResults) > 0) {
					_, err := client.Collection("searches_poi").Doc(strings.ToUpper(query)).Set(ctx, map[string]interface{}{
						"count": 1,
						"value": query,
					})
					if err != nil {
						// Handle any errors in an appropriate way, such as returning them.
						response.WriteErrorResponse(w, err)
					}
				}
			}
		}

		response.Write(w, map[string]interface{}{
			"results": triposoResults,
		}, http.StatusOK)
		return
	
}

// RecentSearch function
func RecentSearch(w http.ResponseWriter, r *http.Request) {
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	poi := r.URL.Query().Get("poi")

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	fmt.Println("Recent Search started")

	defer client.Close()

	errorChannel := make(chan error)
	timeoutChannel := make(chan bool)
	searchChannel := make(chan []interface{})
	var searches []interface{}

	go func() {
		var search *firestore.DocumentIterator
		if poi == "true" {
			search = client.Collection("searches_poi").OrderBy("count", firestore.Desc).Limit(10).Documents(ctx)
		} else {
			search = client.Collection("searches").OrderBy("count", firestore.Desc).Limit(10).Documents(ctx)
		}

		var searchesData []interface{}
		for {
			doc, err := search.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				errorChannel <- err
				break
			}

			searchesData = append(searchesData, doc.Data())
		}
		//fmt.Println(searchesData)
		searchChannel <- searchesData
	}()

	go func() {
		time.Sleep(10 * time.Second)
		timeoutChannel <- true
	}()

	for i := 0; i < 1; i++ {
		select {
		case res := <-searchChannel:
			searches = res
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

	recentSearchData := map[string]interface{}{
		"recent_search": searches,
	}

	response.Write(w, recentSearchData, http.StatusOK)
	return
}

// Popular Locations

func GetPopularLocations(w http.ResponseWriter, r *http.Request) {

	placeChannel := make(chan []triposo.Place)

	var triposoResults []triposo.InternalPlace

	errorChannel := make(chan error)
	timeoutChannel := make(chan bool)

	go func() {

		places, err := triposo.GetLocations("10")
		if err != nil {
			errorChannel <- err
			return
		}
		placeChannel <- *places

	}()

	go func() {
		time.Sleep(10 * time.Second)
		timeoutChannel <- true
	}()

	for i := 0; i < 1; i++ {
		select {
		case res := <-placeChannel:
			triposoResults = FromTriposoPlaces(res, "")
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

	resChannel := make(chan InternalPlaceChannel)

	for index := 0; index < len(triposoResults); index++ {
		go func(index int) {
			if triposoResults[index].CountryID == "United_States" {
				parent, err := triposo.GetLocation(triposoResults[index].ParentID)
				res := new(InternalPlaceChannel)
				parentParam := *parent
				res.Place = FromTriposoPlace(parentParam[0], "")
				res.Index = index
				res.Error = err
				triposoResults[index].CountryName = "United States"

				resChannel <- *res
			} else {
				parent, err := triposo.GetLocation(triposoResults[index].CountryID)
				res := new(InternalPlaceChannel)
				parentParam := *parent
				res.Place = FromTriposoPlace(parentParam[0], "")
				res.Index = index
				res.Error = err
				resChannel <- *res
			}

		}(index)
	}

	for i := 0; i < len(triposoResults); i++ {
		select {
		case res := <-resChannel:
			if res.Error != nil {
				response.WriteErrorResponse(w, res.Error)
				return
			}
			triposoResults[res.Index].ParentName = res.Place.Name
			if len(triposoResults[res.Index].CountryName) == 0 {
				triposoResults[res.Index].CountryName = res.Place.Name
			}
		}
	}

	popularData := map[string]interface{}{
		"results": triposoResults,
	}

	response.Write(w, popularData, http.StatusOK)
	return
}
