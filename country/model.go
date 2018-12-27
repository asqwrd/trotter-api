package country

import (
	"encoding/json"
	"errors"
	"fmt"
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

type InternalVisa struct {
	Visa        Visa        `json:"visa"`
	Passport    Passport    `json:"passport"`
	Vaccination Vaccination `json:"vaccination"`
}

type safetyResponse struct {
	Data SafetyData
}

type SafetyData struct {
	Code      SafetyCode      `json:"code"`
	Situation SafetySituation `json:"situation"`
}

type Safety struct {
	Advice string  `json:"advice"`
	Rating float32 `json:"rating"`
}

type SafetyCode struct {
	Continent string `json:"continent"`
	Country   string `json:"country"`
}

type SafetySituation struct {
	Rating  string `json:"rating"`
	Sources int    `json:"sources"`
}

type Numbers struct {
	All []string `json:"all,omitempty"`
}

type EmergencyNumbers struct {
	Ambulance                 Numbers  `json:"ambulance" firestore:"ambulance"`
	Dispatch                  Numbers  `json:"dispatch" firestore:"dispatch"`
	Fire                      Numbers  `json:"fire" firestore:"fire"`
	Police                    Numbers  `json:"police" firestore:"police"`
	European_emergency_number []string `json:"european_emergency_number"`
	Member112                 bool     `json:"member_112,omitempty" firestore:"member_112"`
}

func FormatEmergencyNumbers(numbers EmergencyNumbers) (e *EmergencyNumbers) {
	member112 := []string{}
	if numbers.Member112 == true {
		member112 = []string{"112"}
	}
	fmt.Println(numbers)
	e = &EmergencyNumbers{
		Ambulance:                 numbers.Ambulance,
		Dispatch:                  numbers.Dispatch,
		Fire:                      numbers.Fire,
		Police:                    numbers.Police,
		European_emergency_number: member112,
	}

	return e
}

func GetCountriesCurrenciesApi() (map[string]interface{}, error) {
	client := http.Client{Timeout: time.Second * 10}
	req, err := http.NewRequest(http.MethodGet, "https://free.currencyconverterapi.com/api/v6/countries", nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Currencies API.")
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Currencies API.")
	}

	resp := &currenciesApiResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Currencies API response.")
	}

	return resp.Results, nil

}

func ConvertCurrency(to string, from string) (map[string]interface{}, error) {
	client := http.Client{Timeout: time.Second * 10}
	req, err := http.NewRequest(http.MethodGet, "https://free.currencyconverterapi.com/api/v6/convert?q="+from+"_"+to+"&compact=n", nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Currency converter API.")
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Currency converter API.")
	}

	resp := &currenciesConvertResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Currency converter API response.")
	}

	return resp.Results[from+"_"+to].(map[string]interface{}), nil

}

func GetVisa(to string, from string) (*visaResponse, error) {
	client := http.Client{Timeout: time.Second * 10}
	req, err := http.NewRequest(http.MethodGet, "https://api.joinsherpa.com/v2/entry-requirements/"+from+"-"+to, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Sherpa API.")
	}
	req.Header.Set("Authorization", "Basic VkRMUUxDYk1tdWd2c09FdGloUTlrZmM2blFvZUdkOm5JWGF4QUxGUFYwSWl3Tk92QkVCckRDTlN3M1NDdjY3UjRVRXZEOXI=")

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Sherpa API.")
	}

	resp := &visaResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Sherpa API response.")
	}

	return resp, nil

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
	var visaData Visa
	if len(visa.Visa) > 0 {
		visaData = visa.Visa[0]
	}
	v = &InternalVisa{
		Visa:        visaData,
		Passport:    *formatPassport(visa.Passport),
		Vaccination: visa.Vaccination,
	}

	return v

}

func GetSafety(countryCode string) (*SafetyData, error) {
	client := http.Client{Timeout: time.Second * 10}
	req, err := http.NewRequest(http.MethodGet, "https://www.reisewarnung.net/api?country="+countryCode, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Safety API.")
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Failed to access the Safety API.")
	}

	resp := &safetyResponse{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("Server experienced an error while parsing Safety API response.")
	}

	return &resp.Data, nil

}

func FormatSafety(rating float32) *string {
	advice := "No safety information is available for this country."
	if rating >= 0 && rating < 1 {
		advice = "Travelling in this country is relatively safe."
	} else if rating >= 1 && rating < 2.5 {
		advice =
			"Travelling in this country is relatively safe. Higher attention is advised when traveling here due to some areas being unsafe."
	} else if rating >= 2.5 && rating < 3.5 {
		advice =
			"This country can be unsafe.  Warnings often relate to specific regions within this country. However, high attention is still advised when moving around. Trotter also recommends traveling to this country with someone who is familiar with the culture and area."
	} else if rating >= 3.5 && rating < 4.5 {
		advice =
			"Travel to this country should be reduced to a necessary minimum and be conducted with good preparation and high attention. If you are not familiar with the area it is recommended you travel with someone who knows the area well."
	} else if rating >= 4.5 {
		advice =
			"It is unsafe to travel to this country.  Trotter advises against traveling here.  You risk high chance of danger to you health and life."
	}
	return &advice
}
