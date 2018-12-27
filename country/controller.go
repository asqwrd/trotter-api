package country

import (
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"time"

	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/location"
	"github.com/asqwrd/trotter-api/places"
	"github.com/asqwrd/trotter-api/response" //"github.com/asqwrd/trotter-api/sygic"
	"github.com/asqwrd/trotter-api/triposo"
	"github.com/gorilla/mux"
	//"github.com/mitchellh/mapstructure"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var citizenCode = "US"
var citizenCountry = "United States"
var currenciesCache map[string]interface{}

func initializeQueryParams(level string) *url.Values {
	qp := &url.Values{}
	qp.Set("level", level)
	return qp
}

func getCurrencies() (map[string]interface{}, error) {
	var errorChannel = make(chan error)
	var currencyChannel = make(chan map[string]interface{})
	var data map[string]interface{}

	go func() {
		res, err := GetCountriesCurrenciesApi()
		if err != nil {
			errorChannel <- err
		}
		currencyChannel <- res
	}()

	for i := 0; i < 1; i++ {
		select {
		case res := <-currencyChannel:
			data = res
		case err := <-errorChannel:
			return nil, err
		}
	}

	return data, nil

}

func GetCountry(w http.ResponseWriter, r *http.Request) {
	runtime.GOMAXPROCS(10)
	fmt.Println("Get Country started");

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

	routeVars := mux.Vars(r)
	countryID := routeVars["countryID"]

	var country triposo.InternalPlace
	var countryColor string
	var popularDestinations []triposo.InternalPlace
	var city_state interface{}

	var plugs []interface{}
	var currency interface{}
	var visa interface{}
	var wg sync.WaitGroup
	resultsChannel := make(chan map[string]interface{})

	var safety Safety

	var emergencyNumbers EmergencyNumbers

	if currenciesCache == nil {
		data, err := getCurrencies()
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		currenciesCache = data
	}

	res, err := triposo.GetLocation(countryID)
	if err != nil {
		resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
		return
	}
	place := *res
	countryRes := places.FromTriposoPlace(place[0], place[0].Type)
	country = countryRes
	//tripname := country.Name
	// //triposoIdRes, err := triposo.GetPlaceByName(tripname)
	// //if err != nil {
	// 	resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
	// 	return
	// }

	/*
		*
		*
		Destination block
		*
		**/
	if country.Type == "country" {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			triposoRes, err := triposo.GetDestination(country.Id, "20")
			if err != nil {
				resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
				return
			}
			resultsChannel <- map[string]interface{}{"result": *triposoRes, "routine": "destination"}
		}(country.Name)
	}

	if country.Type == "city_state" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cityData, err := getCityState(country.Id)
			if err != nil {
				resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
				return
			}
			resultsChannel <- map[string]interface{}{"result": cityData, "routine": "cityData"}
		}()
	}

	/*
		*
		*
		Colors block
		*
		**/
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(country.Image) > 0 {
			colors, err := places.GetColor(country.Image)
			if err != nil {
				resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
				return
			}
			resultsChannel <- map[string]interface{}{"result": colors, "routine": "color"}
		} else {
			var colors places.Colors
			colors.Vibrant = "#c27949"
			resultsChannel <- map[string]interface{}{"result": &colors, "routine": "color"}

		}
	}()

	/*
		*
		*
		Country Code block
		*
		*
		**/

	code, err := client.Collection("countries_code").Doc(country.Name).Get(ctx)
	if err != nil {
		resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
		return
	}

	countryCodeData := code.Data()
	countryCode := countryCodeData["abbreviation"].(string)

	/*
		*
		*
		Visa Block
		*
		*
	*/
	wg.Add(1)
	go func() {
		defer wg.Done()

		visa, err := GetVisa(countryCode, citizenCode)
		if err != nil {
			resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
			return
		}
		resultsChannel <- map[string]interface{}{"result": FormatVisa(*visa), "routine": "visa"}

	}()

	/*
		*
		*
		Safety Block
		*
		*
	*/

	wg.Add(1)
	go func() {
		defer wg.Done()
		safetyRes, err := GetSafety(countryCode)
		if err != nil {
			resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
			return
		}
		resultsChannel <- map[string]interface{}{"result": *safetyRes, "routine": "safety"}
	}()

	/*
		*
		*
		Currency Block
		*
		*
	*/

	wg.Add(1)
	go func() {
		defer wg.Done()
		currency, err := client.Collection("currencies").Doc(countryCode).Get(ctx)
		if err != nil {
			resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
			return
		}

		currencyCodeIdData := currency.Data()
		currencyCodeId := currencyCodeIdData["id"].(string)

		citizenCurrency := currenciesCache["US"].(map[string]interface{})
		var toCurrency map[string]interface{}
		toCurrency = currenciesCache[currencyCodeId].(map[string]interface{})

		currencyData, err := ConvertCurrency(toCurrency["currencyId"].(string), citizenCurrency["currencyId"].(string))
		if err != nil {
			resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
			return
		}
		result := map[string]interface{}{
			"converted_currency": currencyData["val"],
			"converted_unit":     toCurrency,
			"unit":               citizenCurrency,
		}
		resultsChannel <- map[string]interface{}{"result": result, "routine": "currency"}

	}()

	/*
		*
		*
		Emergency numbers block
		*
		*
		**/

	wg.Add(1)
	go func() {
		defer wg.Done()
		numbers, err := client.Collection("emergency_numbers").Doc(countryCode).Get(ctx)
		if err != nil {
			resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
			return
		}
	//	numbersData := numbers.Data()
		var emNumbers EmergencyNumbers
		numbers.DataTo(&emNumbers)
		//var emNumbers EmergencyNumbers
		/*errDecode := mapstructure.Decode(numbersData, &emNumbers)
		if errDecode != nil {
			resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
			return
		}*/

		resultsChannel <- map[string]interface{}{"result": *FormatEmergencyNumbers(emNumbers), "routine": "numbers"}

	}()

	/*
		*
		*
		Plugs block
		*
		*
		**/

	wg.Add(1)
	go func(name string) {
		defer wg.Done()
		var plugsData []interface{}

		iter := client.Collection("plugs").Where("country", "==", name).Documents(ctx)

		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
				break
			}

			plugsData = append(plugsData, doc.Data())
			resultsChannel <- map[string]interface{}{"result": plugsData, "routine": "plugs"}
		}

	}(country.Name)

	go func() {
		wg.Wait()
		close(resultsChannel)
	}()

	var responseData map[string]interface{}

	if country.Type == "city_state" {
		for res := range resultsChannel {
			switch res["routine"] {
			case "cityData":
				city_state = res["result"]
			case "plugs":
				plugs = res["result"].([]interface{})
			case "currency":
				currency = res["result"].(interface{})
			case "visa":
				visa = res["result"].(interface{})
			case "safety":
				ratingRes, err := strconv.ParseFloat(res["result"].(SafetyData).Situation.Rating, 32)
				if err != nil {
					response.WriteErrorResponse(w, err)
					return
				}
				rating := float32(ratingRes)
				safety = Safety{Advice: *FormatSafety(rating), Rating: rating}
			case "numbers":
				emergencyNumbers = res["result"].(EmergencyNumbers)
			case "color":
				colors := res["result"].(*places.Colors)
				if len(colors.Vibrant) > 0 {
					countryColor = colors.Vibrant
				} else if len(colors.Muted) > 0 {
					countryColor = colors.Muted
				} else if len(colors.LightVibrant) > 0 {
					countryColor = colors.LightVibrant
				} else if len(colors.LightMuted) > 0 {
					countryColor = colors.LightMuted
				} else if len(colors.DarkVibrant) > 0 {
					countryColor = colors.DarkVibrant
				} else if len(colors.DarkMuted) > 0 {
					countryColor = colors.DarkMuted
				}
			case "error":
				response.WriteErrorResponse(w, res["result"].(error))
				return
			}
		}
		responseData = map[string]interface{}{
			"city_state":        country,
			"city_state_places": city_state,
			"plugs":             plugs,
			"currency":          currency,
			"color":             countryColor,
			"visa":              visa,
			"safety":            safety,
			"emergency_number":  emergencyNumbers,
		}
	} else {
		for res := range resultsChannel {
			switch res["routine"] {
			case "destination":
				popularDestinations = places.FromTriposoPlaces(res["result"].(interface{}).([]triposo.Place), "city")
			case "plugs":
				plugs = res["result"].([]interface{})
			case "currency":
				currency = res["result"].(interface{})
			case "visa":
				visa = res["result"].(interface{})
			case "safety":
				ratingRes, err := strconv.ParseFloat(res["result"].(SafetyData).Situation.Rating, 32)
				if err != nil {
					response.WriteErrorResponse(w, err)
					return
				}
				rating := float32(ratingRes)
				safety = Safety{Advice: *FormatSafety(rating), Rating: rating}
			case "numbers":
				emergencyNumbers = res["result"].(EmergencyNumbers)
			case "color":
				colors := res["result"].(*places.Colors)
				if len(colors.Vibrant) > 0 {
					countryColor = colors.Vibrant
				} else if len(colors.Muted) > 0 {
					countryColor = colors.Muted
				} else if len(colors.LightVibrant) > 0 {
					countryColor = colors.LightVibrant
				} else if len(colors.LightMuted) > 0 {
					countryColor = colors.LightMuted
				} else if len(colors.DarkVibrant) > 0 {
					countryColor = colors.DarkVibrant
				} else if len(colors.DarkMuted) > 0 {
					countryColor = colors.DarkMuted
				}
			case "error":
				response.WriteErrorResponse(w, res["result"].(error))
				return
			}
		}
		responseData = map[string]interface{}{
			"country":              country,
			"popular_destinations": popularDestinations,
			"plugs":                plugs,
			"currency":             currency,
			"color":                countryColor,
			"visa":                 visa,
			"safety":               safety,
			"emergency_number":     emergencyNumbers,
		}
	}

	response.Write(w, responseData, http.StatusOK)
	return
}

// City State

func getCityState(cityStateID string) (map[string]interface{}, error) {
	urlparams := []string{"sightseeing|sight|topattractions",
		"museums|tours|walkingtours|transport|private_tours|celebrations|hoponhopoff|air|architecture|multiday|touristinfo|forts",
		"amusementparks|golf|iceskating|kayaking|sporttickets|sports|surfing|cinema|zoos",
		"beaches|camping|wildlife|fishing|relaxinapark",
		"eatingout|breakfast|coffeeandcake|lunch|dinner",
		"do|shopping",
		"nightlife|comedy|drinks|dancing|pubcrawl|redlight|musicandshows|celebrations|foodexperiences|breweries|showstheatresandmusic"}

	placeChannel := make(chan triposo.TriposoChannel)

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
	timeoutChannel := make(chan error)

	for i, param := range urlparams {
		go func(param string, i int) {
			place, err := triposo.GetPoiFromLocation(cityStateID, "20", param, i)
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
		time.Sleep(10 * time.Second)
		timeoutChannel <- fmt.Errorf("timeout occured")
	}()

	for i := 0; i < 7; i++ {
		select {
		case see := <-seeChannel:
			placeToSee = places.FromTriposoPlaces(see, "poi")
		case eat := <-eatChannel:
			eatPlaces = places.FromTriposoPlaces(eat, "poi")
		case discover := <-discoverChannel:
			discoverPlaces = places.FromTriposoPlaces(discover, "poi")
		case shop := <-shopChannel:
			shopPlaces = places.FromTriposoPlaces(shop, "poi")
		case relax := <-relaxChannel:
			relaxPlaces = places.FromTriposoPlaces(relax, "poi")
		case play := <-playChannel:
			playPlaces = places.FromTriposoPlaces(play, "poi")
		case nightlife := <-nightlifeChannel:
			nightlifePlaces = places.FromTriposoPlaces(nightlife, "poi")
		case err := <-errorChannel:
			return nil, err
		case timeout := <-timeoutChannel:
			return nil, timeout
		}
	}

	return map[string]interface{}{
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
	}, nil
}
