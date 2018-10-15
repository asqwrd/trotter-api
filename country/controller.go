package country

import (
	"fmt"
	"net/http"
	"github.com/asqwrd/trotter-api/places"
	"net/url"

	"github.com/asqwrd/trotter-api/firebase"
	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/sygic" //"github.com/asqwrd/trotter-api/triposo"
	"github.com/gorilla/mux"
)

var citizenCode = "US"
var citizenCountry = "United States"
var country_codes = trotterFirebase.GetCollection("countries_code")
var emergency_numbers_db = trotterFirebase.GetCollection("emergency_numbers")
var plugs_db = trotterFirebase.GetCollection("plugs")
var currencies_db = trotterFirebase.GetCollection("currencies")
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
	routeVars := mux.Vars(r)
	countryID := routeVars["countryID"]
	var errorChannel = make(chan error)
	var countryChannel = make(chan sygic.PlaceDetail)
	var country places.Place
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
		res, err := sygic.GetPlace(countryID)
		if err != nil {
			errorChannel <- err
			return
		}
		countryChannel <- *res
	}()

	for i := 0; i < 1; i++ {
		select {
		case res := <-countryChannel:
			country = *places.FromSygicPlaceDetail(&res, nil)
		case err := <-errorChannel:
			response.WriteErrorResponse(w, err)
			return
		}
	}

	response.Write(w, country, http.StatusOK)
	return
}
