package types

import (
	"time"

	"firebase.google.com/go/auth"
	"github.com/asqwrd/trotter-api/triposo"
)

//Trip type for trips response
type Trip struct {
	Image        string        `json:"image" firestore:"image"`
	Name         string        `json:"name" firestore:"name"`
	Group        []string      `json:"group" firestore:"group"`
	OwnerID      string        `json:"owner_id" firestore:"owner_id"`
	ID           string        `json:"id" firestore:"id"`
	Color        string        `json:"color" firestore:"color"`
	Destinations []Destination `json:"destinations" firestore:"destinations"`
	ItineraryIDS []string      `json:"itinerary_ids" firestore:"itinerary_ids"`
	UpdatedAt    time.Time     `json:"updated_at" firestore:"updated_at"`
	Travelers    []User        `json:"travelers"`
}

// TripRes Struct for post
type TripRes struct {
	Trip         Trip
	Destinations []Destination
	User         User
	Travelers    []auth.UserInfo
}

// User struct
type User struct {
	DisplayName string `json:"displayName" firestore:"displayName"`
	Email       string `json:"email" firestore:"email"`
	PhoneNumber string `json:"phoneNumber" firestore:"phoneNumber"`
	PhotoURL    string `json:"photoUrl" firestore:"photoUrl"`
	UID         string `json:"uid" firestore:"uid"`
}

//Token struct
type Token struct {
	UID   string `json:"uid" firestore:"uid"`
	Token string `json:"token" firestore:"token"`
}

// Destination struct
type Destination struct {
	Accommodation   Accommodation    `json:"accomodation" firestore:"accomodation"`
	Transportation  Transportation   `json:"transportation" firestore:"transportation"`
	Location        triposo.Location `json:"location" firestore:"location"`
	DestinationID   string           `json:"destination_id" firestore:"destination_id"`
	DestinationName string           `json:"destination_name" firestore:"destination_name"`
	Level           string           `json:"level" firestore:"level"`
	CountryID       string           `json:"country_id" firestore:"country_id"`
	CountryName     string           `json:"country_name" firestore:"country_name"`
	StartDate       int64            `json:"start_date" firestore:"start_date"`
	EndDate         int64            `json:"end_date" firestore:"end_date"`
	ID              string           `json:"id" firestore:"id"`
	ItineraryID     string           `json:"itinerary_id" firestore:"itinerary_id"`
	Image           string           `json:"image" firestore:"image"`
}

// DestinationChannel for routines
type DestinationChannel struct {
	Destinations []Destination
	Index        int
	Error        error
}

// Accommodation type for trip response
type Accommodation struct {
	Type           string        `json:"type" firestore:"type"`
	Source         string        `json:"source" firestore:"source"`
	HotelName      string        `json:"hotel_name" firestore:"hotel_name"`
	Address1       string        `json:"address1" firestore:"address1"`
	Address2       string        `json:"address2" firestore:"address2"`
	CityName       string        `json:"city_name" firestore:"city_name"`
	AdminCode      string        `json:"admin_code" firestore:"admin_code"`
	Country        string        `json:"country" firestore:"country"`
	PostalCode     string        `json:"postal_code" firestore:"postal_code"`
	Lat            string        `json:"lat" firestore:"lat"`
	Lon            string        `json:"lon" firestore:"lon"`
	CheckinDate    string        `json:"checkin_date" firestore:"checkin_date"`
	CheckoutDate   string        `json:"checkout_date" firestore:"checkout_date"`
	TimeZoneID     string        `json:"time_zone_id" firestore:"time_zone_id"`
	Price          string        `json:"price" firestore:"price"`
	Currency       string        `json:"currency" firestore:"currency"`
	NumberOfRooms  string        `json:"number_of_rooms" firestore:"number_of_rooms"`
	ConfirmationNo string        `json:"confirmation_no" firestore:"confirmation_no"`
	RoomType       string        `json:"room_type" firestore:"room_type"`
	PriceDetails   []PriceDetail `json:"price_details" firestore:"price_details"`
}

type PriceDetail struct {
	Type  string `json:"type" firestore:"type"`
	Name  string `json:"name" firestore:"name"`
	Value string `json:"value" firestore:"value"`
	Units string `json:"units" firestore:"units"`
}

type Transportation struct {
	Stops []Stop `json:"stops" firestore:"stops"`
}

type Stop struct {
	Type            string `json:"type" firestore:"type"`
	Source          string `json:"source" firestore:"source"`
	Airline         string `json:"airline,omitempty" firestore:"airline"`
	IataCode        string `json:"iata_code,omitempty" firestore:"iata_code"`
	FlightNumber    string `json:"flight_number,omitempty" firestore:"flight_number"`
	SeatAssignment  string `json:"seat_assignment,omitempty" firestore:"seat_assignment"`
	Origin          string `json:"origin,omitempty" firestore:"origin"`
	OriginName      string `json:"origin_name,omitempty" firestore:"origin_name"`
	OriginCityName  string `json:"origin_city_name,omitempty" firestore:"origin_city_name"`
	OriginAdminCode string `json:"origin_admin_code,omitempty" firestore:"origin_admin_code"`
	OriginCountry   string `json:"origin_country,omitempty" firestore:"origin_country"`
	OriginLat       string `json:"origin_lat,omitempty" firestore:"origin_lat"`
	OriginLon       string `json:"origin_lon,omitempty" firestore:"origin_lon"`

	Destination          string `json:"destination,omitempty" firestore:"destination"`
	DestinationName      string `json:"destination_name,omitempty" firestore:"destination_name"`
	DestinationCityName  string `json:"destination_city_name,omitempty" firestore:"destination_city_name"`
	DestinationAdminCode string `json:"destination_admin_code,omitempty" firestore:"destination_admin_code"`
	DestinationCountry   string `json:"destination_country,omitempty" firestore:"destination_country"`
	DestinationLat       string `json:"destination_lat,omitempty" firestore:"destination_lat"`
	DestinationLon       string `json:"destination_lon,omitempty" firestore:"destination_lon"`

	DepartureDatetime   string `json:"departure_datetime,omitempty" firestore:"departure_datetime"`
	DepartureTimeZoneID string `json:"departure_time_zone_id,omitempty" firestore:"departure_time_zone_id"`

	ArrivalDatetime   string `json:"arrival_datetime,omitempty" firestore:"arrival_datetime"`
	ArrivalTimeZoneID string `json:"arrival_time_zone_id,omitempty" firestore:"arrival_time_zone_id"`
	ConfirmationNo    string `json:"confirmation_no,omitempty" firestore:"confirmation_no"`
	NumberOfPax       string `json:"number_of_pax,omitempty" firestore:"number_of_pax"`
	TicketNumber      string `json:"ticket_number,omitempty" firestore:"ticket_number"`
}
