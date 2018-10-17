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

)



var citizenCode = "US"
var citizenCountry = "United States"
var currenciesCache interface{}

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

func getCurrencies() (*interface{}, error) {
	var errorChannel = make(chan error)
	var currencyChannel = make(chan interface{})
	var data interface{}

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

	return &data, nil

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
	var countryColor places.Colors
	var popularDestinations []triposo.InternalPlace
	var plugsChannel = make(chan []interface{})
	var plugs []interface{}
	if currenciesCache == nil {
		data, err := getCurrencies()
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		fmt.Println("Empty currency")
		currenciesCache = data
	}

	go func() {
		var destinationSubChannel = make(chan []triposo.Place)
		res, err := sygic.GetPlace(countryID)
		if err != nil {
			errorChannel <- err
			return
		}
		var colors places.Colors
		countryRes := places.FromSygicPlaceDetail(res, colors)

		go func(name string, image string) {
			triposoIdRes, err := triposo.GetPlaceByName(name)
			if err != nil {
				errorChannel <- err
				return
			}

			go func(id string){
				triposoRes, err := triposo.GetDestination(id, "20")
				if err != nil {
					errorChannel <- err
					return
				}
				destinationSubChannel <- *triposoRes
			}(triposoIdRes.Id)

			go func(image string){
				colors, err :=places.GetColor(image)
				if err != nil {
					errorChannel <- err
					return
				}
				colorChannel <- *colors
			}(image)
		}(countryRes.Name, countryRes.Image)

		var plugsData []interface{}

		go func(name string){
			iter := client.Collection("plugs").Where("country", "==", name).Documents(ctx)
			
			for{
				doc, err := iter.Next()
				if err == iterator.Done {
					return
				}
				if err != nil {
					errorChannel <- err
					return
				}
				
				plugsData = append(plugsData, doc.Data())
				plugsChannel <- plugsData
			}
			
		}(countryRes.Name)


		
		
		for i := 0; i < 1; i++ {
			select {
			case res := <-destinationSubChannel:
				destinationChannel <- res
			case err := <-errorChannel:
				errorChannel <- err
				return
			}
		}
		

		countryChannel <- *countryRes

		
	}()


	for i := 0; i < 4; i++ {
		select {
		case res := <-countryChannel:
			country = res
		case res := <-colorChannel:
			countryColor = res
		case res := <-destinationChannel:
			popularDestinations = places.FromTriposoPlaces(res)
		case res := <-plugsChannel:
		 	plugs = res
		case err := <-errorChannel:
			response.WriteErrorResponse(w, err)
			return
		}
	}

	country.Colors = countryColor
	if len(countryColor.Vibrant) > 0 {
		country.Color = countryColor.Vibrant
	} else if len(countryColor.Muted) > 0 {
		country.Color = countryColor.Muted
	} else if len(countryColor.LightVibrant) > 0 {
		country.Color = countryColor.LightVibrant
	} else if len(countryColor.LightMuted) > 0 {
		country.Color = countryColor.LightMuted
	} else if len(countryColor.DarkVibrant) > 0 {
		country.Color = countryColor.DarkVibrant
	} else if len(countryColor.DarkMuted) > 0 {
		country.Color = countryColor.DarkMuted
	}

	responseData := map[string]interface{}{
		"country": country,
		"popular_destinations":  popularDestinations,
		"plugs": plugs,
	}


	response.Write(w, responseData, http.StatusOK)
	return
}
