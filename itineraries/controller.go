package itineraries

import (
	//"encoding/json" //"sort"
	//"time"
	"fmt"
	"net/http" //"net/url"
	//"github.com/asqwrd/trotter-api/triposo"
	firebase "firebase.google.com/go" 
	//"github.com/asqwrd/trotter-api/places"
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
	

	/*colors, err := places.GetColor(itinerary.Image)
	if err != nil {
		return nil, err
	}
	if len(colors.Vibrant) > 0 {
		trip.Color = colors.Vibrant
	} else if len(colors.Muted) > 0 {
		trip.Color = colors.Muted
	} else if len(colors.LightVibrant) > 0 {
		trip.Color = colors.LightVibrant
	} else if len(colors.LightMuted) > 0 {
		trip.Color = colors.LightMuted
	} else if len(colors.DarkVibrant) > 0 {
		trip.Color = colors.DarkVibrant 
	} else if len(colors.DarkMuted) > 0 {
		trip.Color = colors.DarkMuted
	}*/
	
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
			itineraryItem.Image = itineraryItem.Poi.Images[0].Sizes.Medium.Url
			itineraryItems = append(itineraryItems,itineraryItem);
			
		}
		day.ItineraryItems = itineraryItems
		days = append(days, day)
	}
	itinerary.Days = days

	itineraryData := map[string]interface{}{
		"itinerary": itinerary,
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