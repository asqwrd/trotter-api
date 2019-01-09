package itineraries

import (
	"github.com/asqwrd/trotter-api/triposo"
)

//Trip type for trips response
type Itinerary struct {
	Days        []Day            `json:"days" firestore:"days"`
	Location    triposo.Location `json:"location" firestore:"location"`
	name        string           `json:"destination_id" firestore:"destination_id"`
	Destination string           `json:"destination_name" firestore:"destination_name"`
	ID          string           `json:"id" firestore:"id"`
}
