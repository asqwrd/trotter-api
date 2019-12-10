package country

import (
	"fmt"
	"net/http"
	"net/url"

	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/places"
	"github.com/asqwrd/trotter-api/response" //"github.com/asqwrd/trotter-api/sygic"
	"github.com/asqwrd/trotter-api/triposo"
	"github.com/gorilla/mux"

	//"github.com/mitchellh/mapstructure"
	"github.com/asqwrd/trotter-api/types"
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
		res, err := GetCountriesCurrenciesAPI()
		if err != nil {
			errorChannel <- err
			return
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

//GetCountry function
func GetCountry(w http.ResponseWriter, r *http.Request) {
	//runtime.GOMAXPROCS(10)
	fmt.Println("Get Country started")

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	var q *url.Values
	args := r.URL.Query()
	q = &args

	userID := q.Get("user_id")

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
	var user types.User
	if len(userID) > 0 {
		docSnap, errGet := client.Collection("users").Doc(userID).Get(ctx)
		if errGet != nil {
			fmt.Println(errGet)
			response.WriteErrorResponse(w, errGet)
			return
		}
		docSnap.DataTo(&user)
	}

	routeVars := mux.Vars(r)
	countryID := routeVars["countryID"]

	var country triposo.InternalPlace
	var countryColor string
	//var popularDestinations []triposo.InternalPlace
	//var cityState interface{}

	var currency interface{}
	var visa interface{}
	routines := 0
	resultsChannel := make(chan map[string]interface{})

	var safety interface{}

	var emergencyNumbers EmergencyNumbers

	if currenciesCache == nil {
		data, err := getCurrencies()
		if err != nil {
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		}
		currenciesCache = data
	}

	res, err := triposo.GetLocation(countryID)
	if err != nil {
		fmt.Println(err)
		resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
		return
	}
	place := *res
	countryRes := places.FromTriposoPlace(place[0], place[0].Type)
	country = countryRes
	fmt.Println(country.Name)

	/*
		*
		*
		Colors block
		*
		**/
	routines++
	go func() {
		//defer wg.Done()
		fmt.Println("color")
		fmt.Println(country.Image)
		if len(country.Image) > 0 {
			colors, err := places.GetColor(country.Image)
			if err != nil {
				fmt.Println("color error")
				fmt.Println(country.Image)
				colorsBackup, errBackup := places.GetColor(country.ImageMedium)
				if errBackup != nil {
					fmt.Println("color backup error")
					fmt.Println(country.ImageMedium)
					resultsChannel <- map[string]interface{}{"result": errBackup, "routine": "error"}
					return
				}
				country.Image = country.ImageMedium
				resultsChannel <- map[string]interface{}{"result": colorsBackup, "routine": "color"}
				return
			}

			resultsChannel <- map[string]interface{}{"result": colors, "routine": "color"}
		} else {
			var colors places.Colors
			colors.Vibrant = "#c27949"
			resultsChannel <- map[string]interface{}{"result": &colors, "routine": "color"}

		}
		fmt.Println("color done")
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
		fmt.Println(err)
		resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
		return
	}
	countryCodeData := code.Data()
	countryCode := countryCodeData["abbreviation"].(string)
	fmt.Println(countryCode)
	/*
		*
		*
		Visa Block
		*
		*
	*/
	if len(userID) > 0 {
		routines++
		go func(user types.User) {
			//defer wg.Done()

			visa, err := GetVisa(countryCode, user.Country)
			if err != nil {
				fmt.Println(err)
				resultsChannel <- map[string]interface{}{"result": nil, "routine": "visa"}
				return
			}
			resultsChannel <- map[string]interface{}{"result": FormatVisa(*visa), "routine": "visa"}
			fmt.Println("visa done")

		}(user)
	}

	/*
		*
		*
		Safety Block
		*
		*
	*/

	routines++
	go func() {
		//defer wg.Done()
		var safetyData SafetyData
		safetyRes, err := client.Collection("safety").Doc(countryCode).Get(ctx)
		if err != nil {
			fmt.Println(err)
			resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
			return
		}
		safetyRes.DataTo(&safetyData)
		resultsChannel <- map[string]interface{}{"result": safetyData, "routine": "safety"}
		fmt.Println("safety done")
	}()

	/*
		*
		*
		Currency Block
		*
		*
	*/
	if len(userID) > 0 {
		//wg.Add(1)
		routines++
		go func(user types.User) {
			//defer wg.Done()
			currency, err := client.Collection("currencies").Doc(countryCode).Get(ctx)
			if err != nil {
				fmt.Println(err)
				resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
				return
			}
			fmt.Println("currency done")

			currencyCodeIDData := currency.Data()
			currencyCodeID := currencyCodeIDData["id"].(string)

			citizenCurrency := currenciesCache[user.Country].(map[string]interface{})
			var toCurrency map[string]interface{}
			toCurrency = currenciesCache[currencyCodeID].(map[string]interface{})
			fmt.Println(citizenCurrency["currencyId"].(string) + "_" + toCurrency["currencyId"].(string))

			currencyData, err := ConvertCurrency(toCurrency["currencyId"].(string), citizenCurrency["currencyId"].(string))
			if err != nil {
				fmt.Println(err)
				result := map[string]interface{}{
					"converted_currency": "",
					"converted_unit":     "",
					"unit":               "",
				}
				resultsChannel <- map[string]interface{}{"result": result, "routine": "currency"}
				fmt.Println("currency done error")
				return
			}
			result := map[string]interface{}{
				"converted_currency": currencyData["val"],
				"converted_unit":     toCurrency,
				"unit":               citizenCurrency,
			}
			resultsChannel <- map[string]interface{}{"result": result, "routine": "currency"}
			fmt.Println("currency done")

		}(user)
	}

	/*
		*
		*
		Emergency numbers block
		*
		*
		**/

	routines++
	go func() {
		//defer wg.Done()
		numbers, err := client.Collection("emergency_numbers").Doc(countryCode).Get(ctx)
		if err != nil {
			fmt.Println(err)
			resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
			return
		}
		//	numbersData := numbers.Data()
		emNumbers := EmergencyNumbers{Dispatch: Numbers{All: []string{}, Fixed: []string{}, GSM: []string{}}, Ambulance: Numbers{All: []string{}, Fixed: []string{}, GSM: []string{}}, Fire: Numbers{All: []string{}, Fixed: []string{}, GSM: []string{}}, Police: Numbers{All: []string{}, Fixed: []string{}, GSM: []string{}}, EuropeanEmergencyNumber: []string{}}
		numbers.DataTo(&emNumbers)

		resultsChannel <- map[string]interface{}{"result": FormatEmergencyNumbers(emNumbers), "routine": "numbers"}
		fmt.Println("numbers done")

	}()

	/*
		*
		*
		Plugs block
		*
		*
		**/

	var plugsData []interface{}

	iter := client.Collection("plugs").Where("country", "==", country.Name).Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			resultsChannel <- map[string]interface{}{"result": err, "routine": "error"}
			break
		}

		plugsData = append(plugsData, doc.Data())
	}
	fmt.Println("plugs done")

	var responseData map[string]interface{}

	for i := 0; i < routines; i++ {
		select {
		case res := <-resultsChannel:
			switch res["routine"] {
			case "currency":
				fmt.Println("currency")
				currency = res["result"].(interface{})
			case "visa":
				fmt.Println("visa")
				if res["result"] != nil {
					visa = res["result"].(interface{})
				} else {
					visa = res["result"]
				}
			case "safety":
				fmt.Println("safety")
				score := res["result"].(SafetyData).Advisory.Score
				safety = Safety{Advice: *FormatSafety(score), Rating: score}
			case "numbers":
				fmt.Println("numbers")
				emergencyNumbers = res["result"].(EmergencyNumbers)
			case "color":
				fmt.Println("color")
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
				fmt.Println("error")
				fmt.Println(res["result"].(error))
				response.WriteErrorResponse(w, res["result"].(error))
				return
			}
		}
	}

	responseData = map[string]interface{}{
		"country":          country,
		"plugs":            plugsData,
		"currency":         currency,
		"color":            countryColor,
		"visa":             visa,
		"safety":           safety,
		"emergency_number": emergencyNumbers,
	}

	response.Write(w, responseData, http.StatusOK)
	return
}
