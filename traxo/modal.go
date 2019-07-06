package traxo

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

const baseTraxoAPI = "https://api.traxo.com/v2/"
const clientID = "6e5a477f"
const clientSecret = "6b80fad42a50250ba7e585620676f9a9"

//Email struct
type Email struct {
	ID          string `json:"id" firestore:"id"`
	MailBoxID   string `json:"mail_box_id" firestore:"mail_box_id"`
	MailboxType string `json:"mailbox_type" firestore:"mailbox_type"`
	Status      string `json:"status" firestore:"status"`
	Source      string `json:"source" firestore:"source"`
	Class       string `json:"class" firestore:"class"`
	UserAddress string `json:"user_address" firestore:"user_address"`
	FromAddress string `json:"from_address" firestore:"from_address"`
	ToAddress   string `json:"to_address" firestore:"to_address"`
	CcAddress   string `json:"cc_address" firestore:"cc_address"`
	Subject     string `json:"subject" firestore:"subject"`
	MsgID       string `json:"msg_id" firestore:"msg_id"`
	MsgDate     string `json:"msg_date" firestore:"msg_date"`
	Created     string `json:"created" firestore:"created"`
	Modified    string `json:"modified" firestore:"modified"`
}

//Info struct
type Info struct {
	Class  int    `json:"class" firestore:"class"`
	Source string `json:"source" firestore:"source"`
}

//Segment struct
type Segment struct {
	Type                 string `json:"type" firestore:"type"`
	Status               string `json:"status" firestore:"status"`
	Source               string `json:"source" firestore:"source"`
	FirstName            string `json:"first_name" firestore:"first_name"`
	LastName             string `json:"last_name" firestore:"last_name"`
	HotelName            string `json:"hotel_name,omitempty" firestore:"hotel_name,omitempty"`
	Address1             string `json:"address1,omitempty" firestore:"address1,omitempty"`
	Address2             string `json:"address2,omitempty" firestore:"address2,omitempty"`
	CityName             string `json:"city_name,omitempty" firestore:"city_name,omitempty"`
	AdminCode            string `json:"admin_code,omitempty" firestore:"admin_code,omitempty"`
	Country              string `json:"country,omitempty" firestore:"country,omitempty"`
	PostalCode           string `json:"postal_code,omitempty" firestore:"postal_code,omitempty"`
	Lat                  string `json:"lat,omitempty" firestore:"lat,omitempty"`
	Lon                  string `json:"lon,omitempty" firestore:"lon,omitempty"`
	CheckinDate          string `json:"checkin_date,omitempty" firestore:"checkin_date,omitempty"`
	CheckOutDate         string `json:"checkout_date,omitempty" firestore:"checkout_date,omitempty"`
	TimeZoneID           string `json:"time_zone_id,omitempty" firestore:"time_zone_id,omitempty"`
	Price                string `json:"price,omitempty" firestore:"price,omitempty"`
	Currency             string `json:"currency,omitempty" firestore:"currency,omitempty"`
	NumberOfRooms        int    `json:"number_of_rooms,omitempty" firestore:"number_of_rooms,omitempty"`
	ConfirmationNo       string `json:"confirmation_no" firestore:"confirmation_no"`
	Phone                string `json:"phone,omitempty" firestore:"phone,omitempty"`
	RoomType             string `json:"room_type,omitempty" firestore:"room_type,omitempty"`
	RoomDescription      string `json:"room_description,omitempty" firestore:"room_description,omitempty"`
	RateDescription      string `json:"rate_description,omitempty" firestore:"rate_description,omitempty"`
	CancellationPolicy   string `json:"cancellation_policy,omitempty" firestore:"cancellation_policy,omitempty"`
	Created              string `json:"created" firestore:"created"`
	Airline              string `json:"airline,omitempty" firestore:"airline,omitempty"`
	IataCode             string `json:"iata_code,omitempty" firestore:"iata_code,omitempty"`
	FlightNumber         string `json:"flight_number,omitempty" firestore:"flight_number,omitempty"`
	OriginAdminCode      string `json:"origin_admin_code,omitempty" firestore:"origin_admin_code,omitempty"`
	OriginLat            string `json:"origin_lat,omitempty" firestore:"origin_lat,omitempty"`
	OriginLon            string `json:"origin_lon,omitempty" firestore:"origin_lon,omitempty"`
	Origin               string `json:"origin,omitempty" firestore:"origin,omitempty"`
	OriginCountry        string `json:"origin_country,omitempty" firestore:"origin_country,omitempty"`
	OriginName           string `json:"origin_name,omitempty" firestore:"origin_name,omitempty"`
	OriginCityName       string `json:"origin_city_name,omitempty" firestore:"origin_city_name,omitempty"`
	Destination          string `json:"destination,omitempty" firestore:"destination,omitempty"`
	DestinationLat       string `json:"destination_lat,omitempty" firestore:"destination_lat,omitempty"`
	DestinationLon       string `json:"destination_lon,omitempty" firestore:"destination_lon,omitempty"`
	DestinationCountry   string `json:"destination_country,omitempty" firestore:"destination_country,omitempty"`
	DestinationName      string `json:"destination_name,omitempty" firestore:"destination_name,omitempty"`
	DestinationCityName  string `json:"destination_city_name,omitempty" firestore:"destination_city_name,omitempty"`
	DestinationAdminCode string `json:"destination_admin_code,omitempty" firestore:"destination_admin_code,omitempty"`
	NormalizedAirline    string `json:"normalized_airline,omitempty" firestore:"normalized_airline,omitempty"`
	DepartureDatetime    string `json:"departure_datetime,omitempty" firestore:"departure_datetime,omitempty"`
	DepartureTimeZoneID  string `json:"departure_time_zone_id,omitempty" firestore:"departure_time_zone_id,omitempty"`
	ArrivalDatetime      string `json:"arrival_datetime,omitempty" firestore:"arrival_datetime,omitempty"`
	ArrivalTimeZoneID    string `json:"arrival_time_zone_id,omitempty" firestore:"arrival_time_zone_id,omitempty"`
}

// Confirmation struct
type Confirmation struct {
	ID          string   `json:"id" firestore:"id"`
	MailBoxID   string   `json:"mail_box_id" firestore:"mail_box_id"`
	MailboxType string   `json:"mailbox_type" firestore:"mailbox_type"`
	Status      string   `json:"status" firestore:"status"`
	Source      string   `json:"source" firestore:"source"`
	Class       string   `json:"class" firestore:"class"`
	UserAddress string   `json:"user_address" firestore:"user_address"`
	FromAddress string   `json:"from_address" firestore:"from_address"`
	ToAddress   string   `json:"to_address" firestore:"to_address"`
	CcAddress   string   `json:"cc_address" firestore:"cc_address"`
	Subject     string   `json:"subject" firestore:"subject"`
	MsgID       string   `json:"msg_id" firestore:"msg_id"`
	MsgDate     string   `json:"msg_date" firestore:"msg_date"`
	Created     string   `json:"created" firestore:"created"`
	Modified    string   `json:"modified" firestore:"modified"`
	Includes    Includes `json:"includes" firestore:"includes"`
}

//Includes struct
type Includes struct {
	Info     Info      `json:"info" firestore:"info"`
	Segments []Segment `json:"segments" firestore:"segments"`
}

//GetEmails function
func GetEmails() (*[]Email, error) {
	client := http.Client{Timeout: time.Second * 30}

	req, err := http.NewRequest(http.MethodGet, baseTraxoAPI+"emails?status=Processed", nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("failed to access the Traxo API")
	}
	//fmt.Println(name)

	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	req.Header.Set("client_id", clientID)
	req.Header.Set("client_secret", clientSecret)

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("failed to access the Traxo API")
	}

	resp := &[]Email{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("server experienced an error while parsing Traxo API response")
	}

	//fmt.Println(resp.Results)
	return resp, nil
}

//GetEmail function
func GetEmail(id string) (*Confirmation, error) {
	client := http.Client{Timeout: time.Second * 30}

	req, err := http.NewRequest(http.MethodGet, baseTraxoAPI+"emails/"+id+"?include=results", nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("failed to access the Traxo API")
	}
	//fmt.Println(name)

	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	req.Header.Set("client_id", clientID)
	req.Header.Set("client_secret", clientSecret)

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errors.New("failed to access the Traxo API")
	}

	resp := &Confirmation{}
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		log.Println(err)
		log.Println(req.URL.String())
		return nil, errors.New("server experienced an error while parsing Traxo API response")
	}

	return resp, nil
}
