package itineraries

import (
	//"encoding/json" //"sort"
	//"time"
	"fmt"
	"net/http" //"net/url"
	"github.com/asqwrd/trotter-api/triposo"
	firebase "firebase.google.com/go" 
	"github.com/asqwrd/trotter-api/places"
	//"cloud.google.com/go/firestore"
	"github.com/asqwrd/trotter-api/response" 
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GetItineraries function
func GetItineraries(w http.ResponseWriter, r *http.Request) {

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	var itineraries []Itinerary
	//colorChannel := make(chan places.ColorChannel)
	daysChannel := make(chan DaysChannel)

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

	iter := client.Collection("itineraries").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		var itinerary Itinerary
		doc.DataTo(&itinerary)
		itineraries = append(itineraries, itinerary)
	}

	for i := 0; i < len(itineraries); i++ {
		go func(index int) {
			var days []Day
			iter := client.Collection("itineraries").Doc(itineraries[index].ID).Collection("days").Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				var day Day
				var itineraryItem ItineraryItem
				var itineraryItems []ItineraryItem
				doc.DataTo(&day)
				iterItems := doc.Ref.Collection("itinerary_items").Documents(ctx)
				for {
					i10ItemsDoc, err := iterItems.Next()
					if err == iterator.Done {
						break
					}
					i10ItemsDoc.DataTo(&itineraryItem)
					itineraryItems = append(itineraryItems,itineraryItem);
					
				}
				day.ItineraryItems = itineraryItems
				days = append(days, day)
			}

			res := new(DaysChannel)
			res.Days = days
			res.Index = index
			daysChannel <- *res
		}(i)
	}
	for i := 0; i < len(itineraries); i++ {
		select {
		case res := <-daysChannel:
			if res.Error != nil {
				response.WriteErrorResponse(w, err)
				return
			}
			itineraries[res.Index].Days = res.Days
		}
	}



	tripsData := map[string]interface{}{
		"itineraries": itineraries,
	}

	fmt.Print("Got Itineraries")

	response.Write(w, tripsData, http.StatusOK)
	return
}

//Private getItinerary funtion
func getItinerary(itineraryID string) (map[string]interface{}, error){
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	errorChannel := make(chan error)
	destinationChannel := make(chan map[string]interface{})
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return nil, err
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	defer client.Close()

	snap, err := client.Collection("itineraries").Doc(itineraryID).Get(ctx)
	if err != nil {
		return nil, err
	}
	var itinerary Itinerary
	snap.DataTo(&itinerary)

	go func(id string){
		parent, err := triposo.GetLocation(id)
		if err != nil {
			errorChannel <- err
		}
		parentParam := *parent
		destination := places.FromTriposoPlace(parentParam[0],parentParam[0].Type);
		colors, err := places.GetColor(destination.Image)
		if err != nil {
			errorChannel <- err
		}

		destinationChannel <- map[string]interface{}{
			"colors": colors,
			"destination": destination,
		}
		

	}(itinerary.Destination)
	
	var days []Day
	iter := client.Collection("itineraries").Doc(itineraryID).Collection("days").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		var day Day
		var itineraryItem ItineraryItem
		var itineraryItems []ItineraryItem
		doc.DataTo(&day)
		iterItems := doc.Ref.Collection("itinerary_items").Documents(ctx)
		for {
			i10ItemsDoc, err := iterItems.Next()
			if err == iterator.Done {
				break
			}
			i10ItemsDoc.DataTo(&itineraryItem)
			if len(itineraryItem.Poi.Images) > 0 {
				itineraryItem.Image = itineraryItem.Poi.Images[0].Sizes.Medium.Url 
			}
			itineraryItems = append(itineraryItems,itineraryItem);
			
		}
		day.ItineraryItems = itineraryItems
		days = append(days, day)
	}
	itinerary.Days = days

	var destination triposo.InternalPlace
	var color string
	for i := 0; i < 1; i++ {
		select{
		case res := <- destinationChannel:
			destination = res["destination"].(triposo.InternalPlace)
			colors := res["colors"].(*places.Colors)
			if len(colors.Vibrant) > 0 {
				color = colors.Vibrant
			} else if len(colors.Muted) > 0 {
				color = colors.Muted
			} else if len(colors.LightVibrant) > 0 {
				color = colors.LightVibrant
			} else if len(colors.LightMuted) > 0 {
				color = colors.LightMuted
			} else if len(colors.DarkVibrant) > 0 {
				color = colors.DarkVibrant 
			} else if len(colors.DarkMuted) > 0 {
				color = colors.DarkMuted
			}
		case err := <- errorChannel:
			return nil, err
		}
	}

	itineraryData := map[string]interface{}{
		"itinerary": itinerary,
		"destination": destination,
		"color": color,
	}
	return itineraryData, err
}

//GetItinerary function
func GetItinerary(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]
	//tripChannel := make(chan Trip)
	itineraryData, err := getItinerary(itineraryID);
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}
	
	response.Write(w, itineraryData, http.StatusOK)
	return
}