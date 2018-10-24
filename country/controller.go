package country

import (
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"sync"

	"github.com/asqwrd/trotter-api/places"

	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/sygic"
	"github.com/asqwrd/trotter-api/triposo"
	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
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

	var country places.Place
	var countryColor string
	var popularDestinations []triposo.InternalPlace

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

	res, err := sygic.GetPlace(countryID)
	if err != nil {
		resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
		return
	}
	countryRes := places.FromSygicPlaceDetail(res)
	country = *countryRes
	tripname := country.Original_name
	if country.Name == "Ireland" {
		tripname = "Republic of Ireland"
	}
	triposoIdRes, err := triposo.GetPlaceByName(tripname)
	if err != nil {
		resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
		return
	}

	/*
		*
		*
		Destination block
		*
		**/
	wg.Add(1)
	go func(name string) {
		defer wg.Done()

		triposoRes, err := triposo.GetDestination(triposoIdRes.Id, "20")
		if err != nil {
			resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
			return
		}
		resultsChannel <- map[string]interface{}{"result": *triposoRes, "routine": "destination"}
		fmt.Println("destinations")
	}(country.Name)

	/*
		*
		*
		Colors block
		*
		**/
	wg.Add(1)
	go func() {
		defer wg.Done()

		colors, err := places.GetColor(country.Image)
		if err != nil {
			resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
			return
		}
		fmt.Println("colors")
		resultsChannel <- map[string]interface{}{"result": colors, "routine": "color"}
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
		fmt.Println("visa")

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
		fmt.Println("safety")
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
		fmt.Println("currency")

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
		numbersData := numbers.Data()
		var emNumbers EmergencyNumbers
		errDecode := mapstructure.Decode(numbersData, &emNumbers)
		if errDecode != nil {
			resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
			return
		}

		fmt.Println("numbers")
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
		fmt.Println("plugs")

	}(country.Name)

	go func() {
		wg.Wait()
		close(resultsChannel)
	}()

	for res := range resultsChannel {
		switch res["routine"] {
		case "destination":
			popularDestinations = places.FromTriposoPlaces(res["result"].(interface{}).([]triposo.Place),"city")
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

	responseData := map[string]interface{}{
		"country":              country,
		"popular_destinations": popularDestinations,
		"plugs":                plugs,
		"currency":             currency,
		"color":                countryColor,
		"visa":                 visa,
		"safety":               safety,
		"emergency_number":     emergencyNumbers,
	}

	response.Write(w, responseData, http.StatusOK)
	return
}
