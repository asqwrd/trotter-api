package country

import (
	"encoding/json"
	"errors"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"time"
)

type currenciesApiResponse struct {
	Status_code int
	Results     interface{}
}

type Passport struct {
	NOT_REQUIRED              string
	ONE                       string
	ONE_PER_ENTRY             string
	SPACE_FOR_STAMP           string
	TWO                       string
	TWO_CONSECUTIVE_PER_ENTRY string
	TWO_PER_ENTRY             string
}

type PassportValidity struct {
	DURATION_OF_STAY                    string
	ONE_MONTH_AFTER_ENTRY               string
	SIX_MONTHS_AFTER_DURATION_OF_STAY   string
	SIX_MONTHS_AFTER_ENTRY              string
	SIX_MONTHS_AT_ENTRY                 string
	THREE_MONTHS_AFTER_DURATION_OF_STAY string
	THREE_MONTHS_AFTER_ENTRY            string
	VALID_AT_ENTRY                      string
	THREE_MONTHS_AFTER_DEPARTURE        string
	SIX_MONTHS_AFTER_DEPARTURE          string
}

func GetCountriesCurrenciesApi() (interface{}, error) {
	client := http.Client{Timeout: time.Second * 5}
	req, err := http.NewRequest(http.MethodGet, "https://free.currencyconverterapi.com/api/v6/countries", nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	resp := &currenciesApiResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Sygic API response.")
	}

	return &resp.Results, nil

}
