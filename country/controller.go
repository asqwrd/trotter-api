package country

import (
	"net/http"
	"github.com/asqwrd/trotter-api/places"
	"net/url"
	"strconv"
	"runtime"
	"fmt"


	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/sygic"
	"github.com/asqwrd/trotter-api/triposo"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	firebase "firebase.google.com/go"
	"github.com/mitchellh/mapstructure"

	//"google.golang.org/grpc"
  //"google.golang.org/grpc/codes"

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
	var errorChannel = make(chan error)
	//var countryChannel = make(chan places.Place)
	var destinationChannel = make(chan []triposo.Place)
	var colorChannel = make(chan places.Colors)
	var country places.Place
	var countryColor string
	var popularDestinations []triposo.InternalPlace
	var plugsChannel = make(chan []interface{})
	var currencyChannel = make(chan map[string]interface{})
	var plugs []interface{}
	var currency interface{}
	var visaChannel = make(chan interface{})
	var visa interface{}


	var safety string
	safetyChannel := make(chan SafetyData)
	emergencyNumbersChannel := make(chan EmergencyNumbers)
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
		errorChannel <- err
		return
	}
	countryRes := places.FromSygicPlaceDetail(res)
	country = *countryRes


	go func(name string, image string) {
		/*
		*
		*
		Destination block
		*
		**/
		tripname := name
		if name == "Ireland" {
			tripname = "Republic of Ireland"
		}
		triposoIdRes, err := triposo.GetPlaceByName(tripname)
		if err != nil {
			errorChannel <- err
			return
		}
		triposoRes, err := triposo.GetDestination(triposoIdRes.Id, "20")
		if err != nil {
			errorChannel <- err
			return
		}
		destinationChannel <- *triposoRes

		/*
		*
		*
		Colors block
		*
		**/

		colors, err :=places.GetColor(image)
		if err != nil {
			errorChannel <- err
			return
		}
		colorChannel <- *colors
		
		/*
		*
		*
		Country Code block
		*
		*
		**/

		code, err := client.Collection("countries_code").Doc(name).Get(ctx)
		if err != nil {
			errorChannel <- err
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
		visa, err := GetVisa(countryCode,citizenCode)
		if err != nil {
			errorChannel <- err
			return
		}
		visaChannel <- FormatVisa(*visa)

		/*
		*
		*
		Safety Block
		*
		*
		*/
		safetyRes, err := GetSafety(countryCode)
		if err != nil {
			errorChannel <- err
			return
		}
		safetyChannel <- *safetyRes

		/*
		*
		*
		Currency Block
		*
		*
		*/
		currency, err := client.Collection("currencies").Doc(countryCode).Get(ctx)
		if err != nil { 
			errorChannel <- err
			return
		}
		
		currencyCodeIdData := currency.Data()
		currencyCodeId := currencyCodeIdData["id"].(string)
		
		citizenCurrency := currenciesCache["US"].(map[string]interface{})
		var toCurrency map[string]interface{}
		toCurrency = currenciesCache[currencyCodeId].(map[string]interface{})
			
		currencyData, err := ConvertCurrency(toCurrency["currencyId"].(string), citizenCurrency["currencyId"].(string))
		if err != nil {
			errorChannel <- err
			return
		}
		result := map[string]interface{}{
			"converted_currency": currencyData["val"],
			"converted_unit": toCurrency,
			"unit": citizenCurrency,
		}
		currencyChannel <- result

			/*
		*
		*
		Emergency numbers block
		*
		*
		**/
		numbers, err := client.Collection("emergency_numbers").Doc(countryCode).Get(ctx)
		if err != nil {
			errorChannel <- err
			return
		}
		numbersData := numbers.Data()
		var emNumbers EmergencyNumbers
		errDecode := mapstructure.Decode(numbersData, &emNumbers)
		if errDecode != nil {
			errorChannel <- err
			return
		}

		fmt.Println(emNumbers)
		emergencyNumbersChannel <- *FormatEmergencyNumbers(emNumbers)
		
		/*
		*
		*
		Plugs block
		*
		*
		**/

		var plugsData []interface{}
		
		iter := client.Collection("plugs").Where("country", "==", name).Documents(ctx)
		
		for{
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				errorChannel <- err
				break
			}
			
			plugsData = append(plugsData, doc.Data())
			plugsChannel <- plugsData
		}

	
		
	}(country.Name, country.Image)


	for i := 0; i < 7; i++ {
		select {
		case res := <-colorChannel:
			if len(res.Vibrant) > 0 {
				countryColor = res.Vibrant
			} else if len(res.Muted) > 0 {
				countryColor = res.Muted
			} else if len(res.LightVibrant) > 0 {
				countryColor = res.LightVibrant
			} else if len(res.LightMuted) > 0 {
				countryColor = res.LightMuted
			} else if len(res.DarkVibrant) > 0 {
				countryColor = res.DarkVibrant
			} else if len(res.DarkMuted) > 0 {
				countryColor = res.DarkMuted
			}
		case res := <-destinationChannel:
			popularDestinations = places.FromTriposoPlaces(res)
		case res := <-plugsChannel:
			plugs = res
		case res := <-currencyChannel:
			currency = res
		case res := <-visaChannel:
			visa = res
		case res := <-safetyChannel:
			rating, err := strconv.ParseFloat(res.Situation.Rating,32)
			if err != nil {
				response.WriteErrorResponse(w, err)
				return
			}
			safety = *FormatSafety(float32(rating))
		case res := <- emergencyNumbersChannel:
			emergencyNumbers = res
		case err := <-errorChannel:
			response.WriteErrorResponse(w, err)
			return

		}
	}

	

	responseData := map[string]interface{}{
		"country": country,
		"popular_destinations":  popularDestinations,
		"plugs": plugs,
		"currency": currency,
		"color": countryColor,
		"visa": visa,
		"safety": safety,
		"emergency_number": emergencyNumbers,
	}

	


	response.Write(w, responseData, http.StatusOK)
	return
}
