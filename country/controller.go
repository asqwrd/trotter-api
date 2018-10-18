package country

import (
	"fmt"
	"net/http"
	"github.com/asqwrd/trotter-api/places"
	"net/url"

	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/sygic"
	"github.com/asqwrd/trotter-api/triposo"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	firebase "firebase.google.com/go"
	//"google.golang.org/grpc"
  //"google.golang.org/grpc/codes"

)



var citizenCode = "US"
var citizenCountry = "United States"
var currenciesCache map[string]interface{}

var passportBlankpages_map = Passport{
	NOT_REQUIRED:              "You do not need to have any blank pages in your passport.",
	ONE:                       "You need at least one blank page in your passport.",
	ONE_PER_ENTRY:             "You need one blank page per entry.",
	SPACE_FOR_STAMP:           "You need space for your passport to be stamped.",
	TWO:                       "You need two blank pages in your passport.",
	TWO_CONSECUTIVE_PER_ENTRY: "You need two consecutive blank pages in your passport",
	TWO_PER_ENTRY:             "You need two blank pages per entry",
}

var passportValidity_map = PassportValidity{
	DURATION_OF_STAY:                    "Your passport must be valid for the duration of your stay in this country.",
	ONE_MONTH_AFTER_ENTRY:               "Your passport must be valid for one month after entering this counrty.",
	SIX_MONTHS_AFTER_DURATION_OF_STAY:   "Your passport must be valid on entry and for six months after the duration of your stay in this country.",
	SIX_MONTHS_AFTER_ENTRY:              "Your passport must be valid on entry and six months after the date of enrty.",
	SIX_MONTHS_AT_ENTRY:                 "Your passport must be valid for at least six months before entering this country.",
	THREE_MONTHS_AFTER_DURATION_OF_STAY: "Your passport must be valid on entry and for three months after the duration of your stay in this country",
	THREE_MONTHS_AFTER_ENTRY:            "Your passport must be valid on entry and for three months after entering this country",
	VALID_AT_ENTRY:                      "Your passport must be valid on entry",
	THREE_MONTHS_AFTER_DEPARTURE:        "Your passport must be valid on entry and three months after your departure date.",
	SIX_MONTHS_AFTER_DEPARTURE:          "Your passport must be valid on entry and six months after your departure date.",
}


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
	var countryChannel = make(chan places.Place)
	var destinationChannel = make(chan []triposo.Place)
	var colorChannel = make(chan places.Colors)
	var country places.Place
	var countryColor string
	var popularDestinations []triposo.InternalPlace
	var plugsChannel = make(chan []interface{})
	var currencyChannel = make(chan map[string]interface{})
	var plugs []interface{}
	var currency interface{}

	if currenciesCache == nil {
		data, err := getCurrencies()
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		currenciesCache = data
	}

	go func() {
		/*destinationSubChannel := make(chan []triposo.Place)
		currencySubChannel := make(chan map[string]interface{})
		errorSubChannel := make(chan error)
		plugsSubChannel := make(chan []interface{})
		colorSubChannel := make(chan places.Colors)*/
		res, err := sygic.GetPlace(countryID)
		if err != nil {
			errorChannel <- err
			return
		}
		countryRes := places.FromSygicPlaceDetail(res)

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
			Currency block
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

			currency, err := client.Collection("currencies").Doc(countryCode).Get(ctx)
			if err != nil { 
				errorChannel <- err
				return
			}
			
			currencyCodeIdData := currency.Data()
			currencyCodeId := currencyCodeIdData["id"].(string)
			
			go func(countryCode string){
				citizenCurrency := currenciesCache["US"].(map[string]interface{})
				var toCurrency map[string]interface{}
					toCurrency = currenciesCache[countryCode].(map[string]interface{})
					
					go func(from map[string]interface{}, to map[string]interface{}) {
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
						fmt.Println("channel")
						currencyChannel <- result
					}(citizenCurrency, toCurrency)
				
			}(currencyCodeId)



			/*
			*
			*
			Plugs block
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
			fmt.Println("channel")
		}(countryRes.Name, countryRes.Image)


		/*for i := 0; i < 4; i++ {
			select {
			case res := <-colorSubChannel:
				colorChannel <- res
			case res := <-destinationSubChannel:
				destinationChannel <- res
			case res := <-plugsSubChannel:
				plugsChannel <- res
			case res := <-currencySubChannel:
				fmt.Println(res)
				currencyChannel <- res
			case err := <-errorSubChannel:
				errorChannel <- err	
			}
		}*/
		

		countryChannel <- *countryRes

		
	}()

	for i := 0; i < 5; i++ {
		select {
		case res := <-countryChannel:
			country = res
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
	}

	


	response.Write(w, responseData, http.StatusOK)
	return
}
