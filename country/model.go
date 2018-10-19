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

var passportValidityMap = map[string]string{
	"DURATION_OF_STAY":                    "Your passport must be valid for the duration of your stay in this country.",
	"ONE_MONTH_AFTER_ENTRY":               "Your passport must be valid for one month after entering this counrty.",
	"SIX_MONTHS_AFTER_DURATION_OF_STAY":   "Your passport must be valid on entry and for six months after the duration of your stay in this country.",
	"SIX_MONTHS_AFTER_ENTRY":              "Your passport must be valid on entry and six months after the date of enrty.",
	"SIX_MONTHS_AT_ENTRY":                 "Your passport must be valid for at least six months before entering this country.",
	"THREE_MONTHS_AFTER_DURATION_OF_STAY": "Your passport must be valid on entry and for three months after the duration of your stay in this country",
	"THREE_MONTHS_AFTER_ENTRY":            "Your passport must be valid on entry and for three months after entering this country",
	"VALID_AT_ENTRY":                      "Your passport must be valid on entry",
	"THREE_MONTHS_AFTER_DEPARTURE":        "Your passport must be valid on entry and three months after your departure date.",
	"SIX_MONTHS_AFTER_DEPARTURE":          "Your passport must be valid on entry and six months after your departure date.",
}

var passportBlankpagesMap = map[string]string{
	"NOT_REQUIRED":              "You do not need to have any blank pages in your passport.",
	"ONE":                       "You need at least one blank page in your passport.",
	"ONE_PER_ENTRY":             "You need one blank page per entry.",
	"SPACE_FOR_STAMP":           "You need space for your passport to be stamped.",
	"TWO":                       "You need two blank pages in your passport.",
	"TWO_CONSECUTIVE_PER_ENTRY": "You need two consecutive blank pages in your passport",
	"TWO_PER_ENTRY":             "You need two blank pages per entry",
}

type currenciesApiResponse struct {
	Status_code int
	Results     map[string]interface{}
}

type currenciesConvertResponse struct {
	Status_code int
	Results     map[string]interface{}
}

type Passport struct {
	BlankPages       string          `json:"blank_pages"`
	PassportValidity string          `json:"passport_validity"`
	Textual          TextualPassport `json:"textual"`
	Currency         Currency        `json:"currency"`
}

type VaccineType struct {
	Type      string `json:"type"`
	Condition string `json:"condition"`
}

type Vaccination struct {
	Recommended []VaccineType `json:"recommended"`
	Required    []VaccineType `json:"required"`
	Risk        []VaccineType `json:"risk"`
}

type Currency struct {
	Arrival string `json:"arrival"`
	Exit    string `json:"exit"`
}

type BlankPages struct {
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

type Textual struct {
	Class string   `json:"class"`
	Text  []string `json:"text"`
}

type PassportValidityTextual struct {
	Textual Textual `json:"textual"`
}

type TextualPassport struct {
	Class            string                  `json:"class"`
	Text             []string                `json:"text"`
	PassportValidity PassportValidityTextual `json:"passport_validity"`
	BlankPages       Textual                 `json:"blank_pages"`
}

type Visa struct {
	Allowed_stay string   `json:"allowed_stay"`
	Notes        []string `json:"notes"`
	Requirement  string   `json:"requirement"`
	Type         string   `json:"type"`
	Textual      Textual  `json:"textual"`
}

type visaResponse struct {
	Passport    Passport    `json:"passport"`
	Visa        []Visa      `json:"visa"`
	Vaccination Vaccination `json:"vaccination"`
}

func GetCountriesCurrenciesApi() (map[string]interface{}, error) {
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

	return resp.Results, nil

}

func ConvertCurrency(to string, from string) (map[string]interface{}, error) {
	client := http.Client{Timeout: time.Second * 5}
	req, err := http.NewRequest(http.MethodGet, "https://free.currencyconverterapi.com/api/v6/convert?q="+from+"_"+to+"&compact=n", nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	resp := &currenciesConvertResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Sygic API response.")
	}

	return resp.Results[from+"_"+to].(map[string]interface{}), nil

}

func GetVisa(to string, from string) (*visaResponse, error) {
	client := http.Client{Timeout: time.Second * 5}
	req, err := http.NewRequest(http.MethodGet, "https://api.joinsherpa.com/v2/entry-requirements/"+from+"-"+to, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}
	req.Header.Set("Authorization", "Basic VkRMUUxDYk1tdWd2c09FdGloUTlrZmM2blFvZUdkOm5JWGF4QUxGUFYwSWl3Tk92QkVCckRDTlN3M1NDdjY3UjRVRXZEOXI=")

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Triposo API.")
	}

	resp := &visaResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Sygic API response.")
	}

	return resp, nil

}

type InternalVisa struct {
	Visa        Visa `json:"visa"`
	Passport    Passport
	Vaccination Vaccination
}

func formatPassport(passport Passport) (p *Passport) {
	blankPages := passportBlankpagesMap[passport.BlankPages]
	if len(blankPages) == 0 {
		blankPages = "To be safe make sure to have at least one blank page in your passport."
	}

	passportValidity := passportValidityMap[passport.PassportValidity]
	if len(passportValidity) == 0 {
		passportValidity = passportValidityMap["VALID_AT_ENTRY"] + " Make sure to check for additional requirements."
	}
	p = &Passport{
		PassportValidity: passportValidity,
		BlankPages:       blankPages,
	}

	return p
}

func FormatVisa(visa visaResponse) (v *InternalVisa) {

	v = &InternalVisa{
		Visa:        visa.Visa[0],
		Passport:    *formatPassport(visa.Passport),
		Vaccination: visa.Vaccination,
	}

	return v

}
