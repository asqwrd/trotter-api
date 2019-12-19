package places

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/location"
	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/sygic"
	"github.com/asqwrd/trotter-api/triposo"
	"github.com/asqwrd/trotter-api/types"
	"github.com/asqwrd/trotter-api/utils"
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
		time.Sleep(30 * time.Second)
		timeoutChannel <- true
	}()

	for i := 0; i < 5; i++ {
		select {
		case city := <-citiesChannel:
			popularCities = append(popularCities, city...)
		case err := <-errorChannel:
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		case timeout := <-timeoutChannel:
			if timeout == true {
				fmt.Println("api timed out")
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

// GetPlaces function
func GetPlaces(w http.ResponseWriter, r *http.Request) {
	args := r.URL.Query()
	q := &args
	placeType := q.Get("type")
	levelID := q.Get("levelId")
	offset := q.Get("offset")
	param := ""
	urlparams := []string{"sightseeing|sight|topattractions|hoponhopoff",
		"museums|tours|walkingtours|transport|private_tours|air|architecture|multiday|touristinfo|forts|showstheatresandmusic",
		"amusementparks|golf|iceskating|kayaking|sporttickets|sports|surfing|cinema|zoos|celebrations|musicandshows",
		"beaches|camping|wildlife|fishing|relaxinapark",
		"eatingout|breakfast|coffeeandcake|lunch|dinner|foodexperiences",
		"do|shopping",
		"nightlife|comedy|drinks|dancing|pubcrawl|redlight|breweries"}

	switch {
	case placeType == "discover":
		param = urlparams[1]
	case placeType == "see":
		param = urlparams[0]
	case placeType == "play":
		param = urlparams[2]
	case placeType == "eat":
		param = urlparams[4]
	case placeType == "nightlife":
		param = urlparams[6]
	case placeType == "shop":
		param = urlparams[5]
	case placeType == "relax":
		param = urlparams[3]

	}
	places, more, err := triposo.GetPoiFromLocationPagination(levelID, "20", param, offset)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	formatResponse := map[string]interface{}{"places": FromTriposoPlaces(*places, "poi"), "more": more}

	response.Write(w, formatResponse, http.StatusOK)
	return

}

// GetPlaceCategory function
func GetPlaceCategory(w http.ResponseWriter, r *http.Request) {
	destinationID := mux.Vars(r)["destinationID"]
	destinationType := r.URL.Query().Get("type")
	query := r.URL.Query().Get("query")
	errorChannel := make(chan error)
	destinationChannel := make(chan triposo.InternalPlace)
	var destination *triposo.InternalPlace
	fmt.Println("Get category")
	go func() {
		destination, err := triposo.GetLocation(destinationID)
		if err != nil {
			//fmt.Println("here")
			errorChannel <- err
			return
		}

		destinationParam := *destination
		destinationRes := FromTriposoPlace(destinationParam[0], destinationType)
		country, err := triposo.GetLocation(destinationRes.CountryID)
		if err != nil {
			errorChannel <- err
			return
		}

		countryParam := *country
		destinationRes.CountryName = countryParam[0].Name

		destinationChannel <- destinationRes

	}()

	for i := 0; i < 1; i++ {
		select {

		case destinationRes := <-destinationChannel:
			destination = &destinationRes

		case err := <-errorChannel:
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return

		}
	}

	googleClient, err := InitGoogle()
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
	}
	var radius uint = 50000
	ctx := context.Background()

	latlng := &maps.LatLng{Lat: destination.Location.Lat, Lng: destination.Location.Lng}
	p := &maps.TextSearchRequest{
		Query:    query,
		Location: latlng,
		Radius:   radius,
	}

	places, err := googleClient.TextSearch(ctx, p)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
	}

	categoryData := map[string]interface{}{
		"places": FromGooglePlaces(places.Results, "poi"),
	}

	response.Write(w, categoryData, http.StatusOK)
	return

}

//GetDestination function
func GetDestination(w http.ResponseWriter, r *http.Request) {
	destinationID := mux.Vars(r)["destinationID"]
	destinationType := r.URL.Query().Get("type")

	destinationChannel := make(chan triposo.InternalPlace)
	colorChannel := make(chan Colors)
	var destination *triposo.InternalPlace
	errorChannel := make(chan error)
	var destinationColor string

	go func() {
		destination, err := triposo.GetLocation(destinationID)
		if err != nil {
			//fmt.Println("here")
			errorChannel <- err
			return
		}

		destinationParam := *destination
		destinationRes := FromTriposoPlace(destinationParam[0], destinationType)
		country, err := triposo.GetLocation(destinationRes.CountryID)
		if err != nil {
			errorChannel <- err
			return
		}

		countryParam := *country
		destinationRes.CountryName = countryParam[0].Name

		if len(destinationRes.Image) == 0 {
			var colors Colors
			colors.Vibrant = "#c27949"
			colorChannel <- colors
		} else {
			colors, err := GetColor(destinationRes.Image)
			if err != nil {
				colorsBackup, errBackup := GetColor(destinationRes.ImageMedium)
				if errBackup != nil {
					errorChannel <- err
					return
				}
				colorChannel <- *colorsBackup
				destinationRes.Image = destinationRes.ImageMedium
				destinationChannel <- destinationRes
				return
			}
			colorChannel <- *colors
		}

		destinationChannel <- destinationRes

	}()

	for i := 0; i < 2; i++ {
		select {

		case destinationRes := <-destinationChannel:
			destination = &destinationRes
		case colorRes := <-colorChannel:
			if len(colorRes.Vibrant) > 0 {
				destinationColor = colorRes.Vibrant
			} else if len(colorRes.Muted) > 0 {
				destinationColor = colorRes.Muted
			} else if len(colorRes.LightVibrant) > 0 {
				destinationColor = colorRes.LightVibrant
			} else if len(colorRes.LightMuted) > 0 {
				destinationColor = colorRes.LightMuted
			} else if len(colorRes.DarkVibrant) > 0 {
				destinationColor = colorRes.DarkVibrant
			} else if len(colorRes.DarkMuted) > 0 {
				destinationColor = colorRes.DarkMuted
			}
		case err := <-errorChannel:
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return

		}
	}

	articles, err := triposo.GetLocationArticles(destinationID)
	if err != nil {
		//fmt.Println("here")
		errorChannel <- err
		return
	}

	destinationData := map[string]interface{}{
		"destination": destination,
		"color":       destinationColor,
		"articles":    articles,
	}

	response.Write(w, destinationData, http.StatusOK)
	return
}

//GetHome function
func GetHome(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got Home")

	placeChannel := make(chan PlaceChannel)

	var cities []triposo.InternalPlace

	errorChannel := make(chan error)

	go func() {

		places, err := triposo.GetLocationType("city", "20")
		//fmt.Println(places)
		res := new(PlaceChannel)
		res.Places = *places
		res.Index = 0
		res.Error = err
		placeChannel <- *res
		close(placeChannel)

	}()

	for res := range placeChannel {
		//fmt.Println(res)
		if res.Error != nil {
			response.WriteErrorResponse(w, res.Error)
			return
		}
		cities = FromTriposoPlaces(res.Places.([]triposo.Place), "city")
	}

	cityParentChannel := make(chan PlaceChannel)
	colorChannel := make(chan ColorChannel)
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
		go func(index int) {

			colors, errColor := GetColor(cities[index].Image)
			if errColor != nil {
				errorChannel <- errColor
				return
			}
			res := new(ColorChannel)
			res.Colors = *colors
			res.Index = index
			res.Error = errColor

			colorChannel <- *res

		}(i)
	}

	for i := 0; i < len(cities)*2; i++ {
		select {
		case res := <-cityParentChannel:
			if res.Error != nil {
				fmt.Println(res.Error)
				response.WriteErrorResponse(w, res.Error)
				return
			}
			cities[res.Index].CountryName = res.Places.([]triposo.Place)[0].Name
			if cities[res.Index].CountryID == "United_States" {
				cities[res.Index].CountryName = "United States"
			}
			cities[res.Index].ParentName = res.Places.([]triposo.Place)[0].Name

		case res := <-colorChannel:
			if res.Error != nil {
				response.WriteErrorResponse(w, res.Error)
				return
			}
			colors := res.Colors
			i := res.Index
			if len(colors.Vibrant) > 0 {
				cities[i].Color = colors.Vibrant
			} else if len(colors.Muted) > 0 {
				cities[i].Color = colors.Muted
			} else if len(colors.LightVibrant) > 0 {
				cities[i].Color = colors.LightVibrant
			} else if len(colors.LightMuted) > 0 {
				cities[i].Color = colors.LightMuted
			} else if len(colors.DarkVibrant) > 0 {
				cities[i].Color = colors.DarkVibrant
			} else if len(colors.DarkMuted) > 0 {
				cities[i].Color = colors.DarkMuted
			}
		case err := <-errorChannel:
			response.WriteErrorResponse(w, err)
			return
		}
	}

	homeData := map[string]interface{}{
		"popular_cities": cities,
	}

	response.Write(w, homeData, http.StatusOK)
	fmt.Println("home done")
	return
}

//UpdatePoiImage func
func UpdatePoiImage(w http.ResponseWriter, r *http.Request) {
	poiID := mux.Vars(r)["poiId"]
	itineraryItemID := mux.Vars(r)["itineraryItemId"]
	itineraryID := mux.Vars(r)["itineraryId"]
	dayID := mux.Vars(r)["dayId"]
	var poiColor string
	poiChannel := make(chan InternalPlaceChannel)
	colorChannel := make(chan ColorChannel)

	ctx := context.Background()
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	googleClient, err := InitGoogle()
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	go func() {
		defer close(poiChannel)
		placeDetail := &maps.PlaceDetailsRequest{
			PlaceID: poiID,
		}
		place, err := googleClient.PlaceDetails(ctx, placeDetail)
		if err != nil {
			fmt.Println(err)
			poiChannel <- InternalPlaceChannel{Error: err}
			return
		}
		photo := "https://maps.googleapis.com/maps/api/place/photo?maxwidth=1280&photoreference=" + place.Photos[0].PhotoReference + "&key=" + googleAPI
		go func(photo string) {
			defer close(colorChannel)
			if len(photo) == 0 {
				var colors Colors
				colors.Vibrant = "#c27949"
				colorChannel <- ColorChannel{Colors: colors}
			} else {
				colors, err := GetColor(photo)
				if err != nil {
					var color Colors
					color.Vibrant = "#c27949"
					colorChannel <- ColorChannel{Colors: color}
					return
				}
				colorChannel <- ColorChannel{Colors: *colors}

			}
		}(photo)
		poi := FromGooglePlace(place, "poi")

		poiChannel <- InternalPlaceChannel{Place: poi}
	}()

	var newPoi triposo.InternalPlace
	fmt.Println(itineraryItemID)
	for res := range poiChannel {
		if res.Error != nil {
			response.WriteErrorResponse(w, res.Error)
			return
		}
		poi := res.Place
		_, err := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(itineraryItemID).Set(ctx, map[string]interface{}{
			"poi": poi,
		}, firestore.MergeAll)
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}

		newPoi = poi
	}

	for res := range colorChannel {
		color := res.Colors
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
	}

	poiData := map[string]interface{}{
		"poi":   newPoi,
		"color": poiColor,
	}

	response.Write(w, poiData, http.StatusOK)
	return

}

//GetPoi func
func GetPoi(w http.ResponseWriter, r *http.Request) {
	poiID := mux.Vars(r)["poiID"]
	googlePlace := r.URL.Query().Get("googlePlace")
	locationID := r.URL.Query().Get("locationId")
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
			if err != nil {
				errorChannel <- err
				return
			}
			r := &maps.PlaceDetailsRequest{
				PlaceID: poiID,
			}
			place, err := googleClient.PlaceDetails(ctx, r)
			if err != nil {
				errorChannel <- err
				return
			}
			photo := "https://maps.googleapis.com/maps/api/place/photo?maxwidth=1280&photoreference=" + place.Photos[0].PhotoReference + "&key=" + googleAPI
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

			if len(poiRes.Image) == 0 {
				var colors Colors
				colors.Vibrant = "#c27949"
				colorChannel <- colors
			} else {
				colors, err := GetColor(poiRes.Image)
				if err != nil {
					colorsBackup, errBackup := GetColor(poiRes.ImageMedium)
					if errBackup != nil {
						errorChannel <- err
					}
					colorChannel <- *colorsBackup
					poiRes.Image = poiRes.ImageMedium
					poiChannel <- poiRes
					return
				}
				colorChannel <- *colors
				poiChannel <- poiRes
				return
			}
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
				poi.LocationID = locationID
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
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		case timeout := <-timeoutChannel:
			if timeout == true {
				fmt.Println("api timeout")
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

//GetPark function
func GetPark(w http.ResponseWriter, r *http.Request) {
	parkID := mux.Vars(r)["parkID"]

	parkChannel := make(chan triposo.InternalPlace)
	colorChannel := make(chan Colors)
	var park *triposo.InternalPlace

	var pois map[string]interface{}

	poiChannel := make(chan triposo.Channel)
	errorChannel := make(chan error)
	timeoutChannel := make(chan bool)
	var parkColor string

	go func() {
		places, more, err := triposo.GetPoiFromLocation(parkID, "20", "", 0)
		if err != nil {
			errorChannel <- err
			return
		}
		var res triposo.Channel
		res.Places = *places
		res.More = more
		poiChannel <- res
	}()

	go func() {
		parkData, err := triposo.GetLocation(parkID)
		if err != nil {
			errorChannel <- err
			return
		}

		parkParam := *parkData
		parkRes := FromTriposoPlace(parkParam[0], "national_park")

		if len(parkRes.Image) == 0 {
			var colors Colors
			colors.Vibrant = "#c27949"
			colorChannel <- colors
		} else {
			colors, err := GetColor(parkRes.Image)
			if err != nil {
				colorsBackup, errBackup := GetColor(parkRes.ImageMedium)
				if errBackup != nil {
					errorChannel <- err
					return
				}
				colorChannel <- *colorsBackup
				parkRes.Image = parkRes.ImageMedium
				parkChannel <- parkRes
				return
			}
			colorChannel <- *colors
		}

		parkChannel <- parkRes

	}()

	go func() {
		time.Sleep(30 * time.Second)
		timeoutChannel <- true
	}()

	for i := 0; i < 3; i++ {
		select {
		case poi := <-poiChannel:
			pois = map[string]interface{}{"places": FromTriposoPlaces(poi.Places, "poi"), "more": poi.More}
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
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		case timeout := <-timeoutChannel:
			if timeout == true {
				fmt.Println("api timeout")
				response.WriteErrorResponse(w, fmt.Errorf("api timeout"))
				return
			}
		}
	}

	parkData := map[string]interface{}{
		"park":  park,
		"color": parkColor,

		"pois":          &pois,
		"poi_locations": location.FromTriposoPlaces(pois["places"].([]triposo.InternalPlace)),
	}

	response.Write(w, parkData, http.StatusOK)
	return
}

// Search function
func Search(w http.ResponseWriter, r *http.Request) {
	query := mux.Vars(r)["query"]
	latq := r.URL.Query().Get("lat")
	lngq := r.URL.Query().Get("lng")

	lat, _ := strconv.ParseFloat(latq, 64)
	lng, _ := strconv.ParseFloat(lngq, 64)

	errorChannel := make(chan error)
	timeoutChannel := make(chan bool)
	addQuery := make(chan bool)
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	if len(latq) == 0 && len(lngq) == 0 {
		typeparams := []string{"island", "city", "city_state", "region"}
		placeChannel := make(chan PlaceChannel)

		triposoResults := []triposo.InternalPlace{}

		islandChannel := make(chan []triposo.Place)
		cityChannel := make(chan []triposo.Place)
		cityStateChannel := make(chan []triposo.Place)
		//parkChannel := make(chan []triposo.Place)
		regionChannel := make(chan []triposo.Place)

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
				fmt.Println(errSearch)
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

				// case res.Index == 3:
				// 	parkChannel <- res.Places.([]triposo.Place)
				case res.Index == 3:
					regionChannel <- res.Places.([]triposo.Place)
				}
			}

		}()

		go func() {
			time.Sleep(30 * time.Second)
			timeoutChannel <- true
		}()

		for i := 0; i < 4; i++ {
			select {
			case res := <-islandChannel:
				triposoResults = append(triposoResults, FromTriposoPlaces(res, "island")...)
			case res := <-cityChannel:
				triposoResults = append(triposoResults, FromTriposoPlaces(res, "city")...)
			case res := <-cityStateChannel:
				triposoResults = append(triposoResults, FromTriposoPlaces(res, "city_state")...)
			// case res := <-parkChannel:
			// 	triposoResults = append(triposoResults, FromTriposoPlaces(res, "national_park")...)
			case res := <-regionChannel:
				triposoResults = append(triposoResults, FromTriposoPlaces(res, "region")...)
			case err := <-errorChannel:
				fmt.Println(err)
				response.WriteErrorResponse(w, err)
				return
			case timeout := <-timeoutChannel:
				if timeout == true {
					fmt.Println("api timeout")
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
					fmt.Println(res.Error)
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
						fmt.Println(err)
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
	}
	triposoResults := []triposo.InternalPlace{}
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
			fmt.Println(errSearch)
			response.WriteErrorResponse(w, errSearch)
		}
		addQuery <- false

	}()

	googlePlaceChannel := make(chan triposo.InternalPlace)
	googleClient, err := InitGoogle()
	if err != nil {
		errorChannel <- err
		return
	}
	latlng := &maps.LatLng{Lat: lat, Lng: lng}
	re := &maps.PlaceAutocompleteRequest{
		Input:    query,
		Location: latlng,
		Radius:   50000,
	}
	places, err := googleClient.PlaceAutocomplete(ctx, re)
	if err != nil {
		errorChannel <- err
		return
	}
	for i := 0; i < len(places.Predictions); i++ {
		go func(placeID string) {
			r := &maps.PlaceDetailsRequest{
				PlaceID: placeID,
			}
			place, err := googleClient.PlaceDetails(ctx, r)
			googlePlaceChannel <- FromGooglePlace(place, "poi")
			if err != nil {
				errorChannel <- err
				return
			}
		}(places.Predictions[i].PlaceID)
	}

	go func() {
		time.Sleep(30 * time.Second)
		timeoutChannel <- true
	}()

	for i := 0; i < len(places.Predictions); i++ {
		select {
		case res := <-googlePlaceChannel:
			triposoResults = append(triposoResults, res)
		case err := <-errorChannel:
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		case timeout := <-timeoutChannel:
			if timeout == true {
				fmt.Println("api timeout")
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
					fmt.Println(err)
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

//ThingsToDo function
func ThingsToDo(w http.ResponseWriter, r *http.Request) {
	var q *url.Values
	args := r.URL.Query()
	q = &args

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	destinationsChannel := make(chan types.DoChannel)

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	iter := client.Collection("trips").Where("group", "array-contains", q.Get("user_id")).Documents(ctx)
	currentTime := time.Now().Unix()
	destinations := []map[string]interface{}{}
	go func(iter *firestore.DocumentIterator) {
		defer close(destinationsChannel)
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				fmt.Println(err)
				response.WriteErrorResponse(w, err)
				return
			}
			var trip types.Trip
			doc.DataTo(&trip)

			iterDestinations := client.Collection("trips").Doc(trip.ID).Collection("destinations").Where("end_date", ">=", currentTime).Documents(ctx)
			for {
				chanRes := types.DoChannel{}
				destinationsDoc, errDestinations := iterDestinations.Next()
				if errDestinations == iterator.Done {
					break
				}
				if errDestinations != nil {
					fmt.Println(errDestinations)
					chanRes.Error = errDestinations
					destinationsChannel <- chanRes
					return
				}
				var destination types.Destination
				destinationsDoc.DataTo(&destination)
				fmt.Println(destination.Image)
				colorRes, errColor := GetColor(destination.Image)
				if errColor != nil {
					fmt.Println(errColor)
					chanRes.Error = errColor
					destinationsChannel <- chanRes
					return
				}
				var destinationColor string = ""
				if len(colorRes.Vibrant) > 0 {
					destinationColor = colorRes.Vibrant
				} else if len(colorRes.Muted) > 0 {
					destinationColor = colorRes.Muted
				} else if len(colorRes.LightVibrant) > 0 {
					destinationColor = colorRes.LightVibrant
				} else if len(colorRes.LightMuted) > 0 {
					destinationColor = colorRes.LightMuted
				} else if len(colorRes.DarkVibrant) > 0 {
					destinationColor = colorRes.DarkVibrant
				} else if len(colorRes.DarkMuted) > 0 {
					destinationColor = colorRes.DarkMuted
				}

				chanRes.Destination = map[string]interface{}{
					"destination": destination,
					"color":       destinationColor,
				}

				destinationsChannel <- chanRes

			}

			iterDestinationsNum := client.Collection("trips").Doc(trip.ID).Collection("destinations").Where("num_of_days", ">", 0).Documents(ctx)
			for {
				chanRes := types.DoChannel{}
				destinationsDoc, errDestinations := iterDestinationsNum.Next()
				if errDestinations == iterator.Done {
					break
				}
				if errDestinations != nil {
					fmt.Println(errDestinations)
					chanRes.Error = errDestinations
					destinationsChannel <- chanRes
					return
				}
				var destination types.Destination
				destinationsDoc.DataTo(&destination)

				colorRes, errColor := GetColor(destination.Image)
				if errColor != nil {
					fmt.Println(errColor)
					chanRes.Error = errColor
					destinationsChannel <- chanRes
					return

				}
				var destinationColor string = ""
				if len(colorRes.Vibrant) > 0 {
					destinationColor = colorRes.Vibrant
				} else if len(colorRes.Muted) > 0 {
					destinationColor = colorRes.Muted
				} else if len(colorRes.LightVibrant) > 0 {
					destinationColor = colorRes.LightVibrant
				} else if len(colorRes.LightMuted) > 0 {
					destinationColor = colorRes.LightMuted
				} else if len(colorRes.DarkVibrant) > 0 {
					destinationColor = colorRes.DarkVibrant
				} else if len(colorRes.DarkMuted) > 0 {
					destinationColor = colorRes.DarkMuted
				}

				chanRes.Destination = map[string]interface{}{
					"destination": destination,
					"color":       destinationColor,
				}

				destinationsChannel <- chanRes
			}

		}
	}(iter)

	for res := range destinationsChannel {
		if res.Error != nil {
			response.WriteErrorResponse(w, res.Error)
			return
		}
		destinations = append(destinations, res.Destination)
	}

	response.Write(w, map[string]interface{}{
		"destinations": utils.UniqueDestinationsSlice(destinations),
	}, http.StatusOK)
	return

}

//NearBy function
func NearBy(w http.ResponseWriter, r *http.Request) {
	var q *url.Values
	args := r.URL.Query()
	q = &args
	latq := q.Get("lat")
	lngq := q.Get("lng")
	placeType := q.Get("type")
	//keywords := q.Get("keywords")
	lat, errlat := strconv.ParseFloat(latq, 64)
	lng, errlng := strconv.ParseFloat(lngq, 64)
	if errlng != nil {
		fmt.Println(errlng)
		response.WriteErrorResponse(w, errlng)
		return
	}
	if errlat != nil {
		fmt.Println(errlat)
		response.WriteErrorResponse(w, errlat)
		return
	}

	ctx := context.Background()

	googleClient, err := InitGoogle()
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
	}
	latlng := &maps.LatLng{Lat: lat, Lng: lng}
	//var radius uint = 5000

	p := &maps.NearbySearchRequest{
		Type:     maps.PlaceType(placeType),
		Location: latlng,
		//Radius:   radius,
		OpenNow: true,
		//Keyword: keywords,
		RankBy: maps.RankByDistance,
	}

	places, err := googleClient.NearbySearch(ctx, p)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
	}

	response.Write(w, map[string]interface{}{
		"places": FromGooglePlaces(places.Results, "poi"),
	}, http.StatusOK)

	return

}

// SearchGoogle function
func SearchGoogle(w http.ResponseWriter, r *http.Request) {
	query := mux.Vars(r)["query"]
	latq := r.URL.Query().Get("lat")
	lngq := r.URL.Query().Get("lng")
	isNear := r.URL.Query().Get("isNear")
	pageToken := r.URL.Query().Get("nextPageToken")

	fmt.Println("isNear")
	fmt.Println(isNear)

	lat, errlat := strconv.ParseFloat(latq, 64)
	lng, errlng := strconv.ParseFloat(lngq, 64)
	if errlng != nil {
		fmt.Println(errlng)
		response.WriteErrorResponse(w, errlng)
		return
	}
	if errlat != nil {
		fmt.Println(errlat)
		response.WriteErrorResponse(w, errlat)
		return
	}

	addQuery := make(chan bool)
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()
	triposoResults := []triposo.InternalPlace{}

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
			fmt.Println(errSearch)
			response.WriteErrorResponse(w, errSearch)
		}
		addQuery <- false

	}()

	googleClient, err := InitGoogle()
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
	}
	latlng := &maps.LatLng{Lat: lat, Lng: lng}
	var radius uint = 50000
	if len(isNear) > 0 {
		radius = 5000
	}
	fmt.Println("radius")
	fmt.Println(radius)
	p := &maps.TextSearchRequest{
		Query:    query,
		Location: latlng,
		Radius:   radius,
	}
	if len(pageToken) > 0 {
		p.PageToken = pageToken
	}
	places, err := googleClient.TextSearch(ctx, p)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
	}
	for i := 0; i < len(places.Results); i++ {
		triposoResults = append(triposoResults, FromGooglePlaceSearch(places.Results[i], "poi"))
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
					fmt.Println(err)
					response.WriteErrorResponse(w, err)
				}
			}
		}
	}

	response.Write(w, map[string]interface{}{
		"results":       triposoResults,
		"nextPageToken": places.NextPageToken,
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
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
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
		time.Sleep(30 * time.Second)
		timeoutChannel <- true
	}()

	for i := 0; i < 1; i++ {
		select {
		case res := <-searchChannel:
			searches = res
		case err := <-errorChannel:
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		case timeout := <-timeoutChannel:
			if timeout == true {
				fmt.Println("api timeout")
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

//GetPopularLocations function
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
		time.Sleep(30 * time.Second)
		timeoutChannel <- true
	}()

	for i := 0; i < 1; i++ {
		select {
		case res := <-placeChannel:
			triposoResults = FromTriposoPlaces(res, "")
		case err := <-errorChannel:
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		case timeout := <-timeoutChannel:
			if timeout == true {
				fmt.Println("api timeout")
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
				fmt.Println(res.Error)
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
