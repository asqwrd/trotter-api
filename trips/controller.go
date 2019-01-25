package trips

import (
	"encoding/json" //"sort"
	"time"
	"fmt"
	"net/http" //"net/url"
	//"github.com/asqwrd/trotter-api/triposo"
	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/places" 
	"cloud.google.com/go/firestore" 
	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/itineraries"
	"github.com/asqwrd/trotter-api/types/trips"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GetTrips function
func GetTrips(w http.ResponseWriter, r *http.Request) {

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	var trips []triptypes.Trip
	colorChannel := make(chan places.ColorChannel)
	destinationChannel := make(chan triptypes.DestinationChannel)

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

	iter := client.Collection("trips").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		var trip triptypes.Trip
		doc.DataTo(&trip)
		trips = append(trips, trip)
	}
	for i := 0; i < len(trips); i++ {
		go func(index int) {

			colors, err := places.GetColor(trips[index].Image)
				if err != nil {
					response.WriteErrorResponse(w, err);
				}

				res := new(places.ColorChannel)
				res.Colors = *colors
				res.Index = index
				colorChannel <- *res
				
		}(i)
		go func(index int){
			var dest []triptypes.Destination
			iter := client.Collection("trips").Doc(trips[index].ID).Collection("destinations").Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					response.WriteErrorResponse(w, err)
					return
				}
				var destination triptypes.Destination
				doc.DataTo(&destination)
				dest = append(dest, destination)
			}
			res := new(triptypes.DestinationChannel)
			res.Destinations = dest
			res.Index = index
			destinationChannel <- *res
		}(i)
	}
	for i:=0; i < len(trips)*2; i++ {
		select{
		case colors := <- colorChannel:
			if len(colors.Colors.Vibrant) > 0 {
				trips[colors.Index].Color = colors.Colors.Vibrant
			} else if len(colors.Colors.Muted) > 0 {
				trips[colors.Index].Color = colors.Colors.Muted
			} else if len(colors.Colors.LightVibrant) > 0 {
				trips[colors.Index].Color = colors.Colors.LightVibrant
			} else if len(colors.Colors.LightMuted) > 0 {
				trips[colors.Index].Color = colors.Colors.LightMuted
			} else if len(colors.Colors.DarkVibrant) > 0 {
				trips[colors.Index].Color = colors.Colors.DarkVibrant
			} else if len(colors.Colors.DarkMuted) > 0 {
				trips[colors.Index].Color = colors.Colors.DarkMuted
			}
		case des := <- destinationChannel:
			trips[des.Index].Destinations = des.Destinations
		}
	}

	tripsData := map[string]interface{}{
		"trips": trips,
	}

	fmt.Print("Got trips");

	response.Write(w, tripsData, http.StatusOK)
	return
}

// CreateTrip function
func CreateTrip(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var trip triptypes.TripRes
	destinationChannel := make(chan string)
	err := decoder.Decode(&trip)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

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

	doc, _, errCreate := client.Collection("trips").Add(ctx, trip.Trip)
	if errCreate != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errCreate)
		response.WriteErrorResponse(w, errCreate)
	}

	_, err2 := client.Collection("trips").Doc(doc.ID).Set(ctx, map[string]interface{}{
		"id": doc.ID,
	},firestore.MergeAll)
	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		response.WriteErrorResponse(w, err2)
	}

	//Adding destinations

	for i:=0; i < len(trip.Destinations); i++ {
		go func(index int, tripID string){
			destDoc, _, errCreate := client.Collection("trips").Doc(tripID).Collection("destinations").Add(ctx, trip.Destinations[index])
			if errCreate != nil {
				// Handle any errors in an appropriate way, such as returning them.
				fmt.Println(errCreate)
				response.WriteErrorResponse(w, errCreate)
			}

			_, err2 := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destDoc.ID).Set(ctx, map[string]interface{}{
				"id": destDoc.ID,
			},firestore.MergeAll)
			if err2 != nil {
				// Handle any errors in an appropriate way, such as returning them.
				response.WriteErrorResponse(w, err2)
				return
			}
			destinationChannel <- destDoc.ID
		}(i, doc.ID)
	}
	var destIDS []string
	for i:=0; i < len(trip.Destinations); i++ {
		select{
		case res := <- destinationChannel:
			destIDS = append(destIDS,res)
		}
	}
	
	id := doc.ID
	tripData, err := getTrip(id)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	response.Write(w, tripData, http.StatusOK)
	return
}

//Private getTrip funtion
func getTrip(tripID string) (map[string]interface{}, error){
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	fmt.Println("Got Trips")
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return nil, err
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	defer client.Close()

	snap, err := client.Collection("trips").Doc(tripID).Get(ctx)
	if err != nil {
		return nil, err
	}
	var trip triptypes.Trip
	snap.DataTo(&trip)
	

	colors, err := places.GetColor(trip.Image)
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
	}

	var dest []triptypes.Destination
	iter := client.Collection("trips").Doc(tripID).Collection("destinations").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var destination triptypes.Destination
		doc.DataTo(&destination)
		dest = append(dest, destination)
	}
	tripData := map[string]interface{}{
		"trip": trip,
		"destinations": dest,
	}
	return tripData, err
}

// GetTrip function
func GetTrip(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	//tripChannel := make(chan Trip)
	tripData, err := getTrip(tripID);
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}
	
	response.Write(w, tripData, http.StatusOK)
	return
}

// UpdateTrip function
func UpdateTrip(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	decoder := json.NewDecoder(r.Body)
	var trip map[string]interface{}
	err := decoder.Decode(&trip)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

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
	fmt.Println(trip)

	_, err2 := client.Collection("trips").Doc(tripID).Set(ctx, trip,firestore.MergeAll)

	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		response.WriteErrorResponse(w, err2)	
		return
	}

	tripData := map[string]interface{}{
		"success": true,
	}

	response.Write(w, tripData, http.StatusOK)	
	return
}

// UpdateDestination function
func UpdateDestination(w http.ResponseWriter, r *http.Request) {
	destinationID := mux.Vars(r)["destinationId"]
	tripID := mux.Vars(r)["tripId"]
	decoder := json.NewDecoder(r.Body)
	var destination map[string]interface{}
	errorChannel := make(chan error)
	dayIdsChannel := make(chan string)

	err := decoder.Decode(&destination)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

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


	_, err2 := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Set(ctx, destination,firestore.MergeAll)

	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		response.WriteErrorResponse(w, err2)	
		return
	}
	 itineraryID := destination["itinerary_id"].(string)
	if len(itineraryID) > 0 {
		_,errUpdateI10 := client.Collection("itineraries").Doc(itineraryID).Set(ctx,map[string]interface{}{
			"end_date": destination["end_date"],
			"start_date": destination["start_date"],
		},firestore.MergeAll)

		if errUpdateI10 != nil {
			response.WriteErrorResponse(w, err2)	
			return
		}
		var daysCount = 0	
		endtm := time.Unix(int64(destination["end_date"].(float64)), 0)
		starttm := time.Unix(int64(destination["start_date"].(float64)), 0)

		diff := endtm.Sub(starttm)
		daysCount = int(diff.Hours()/24) + 1 //include first day
		var itinerariesItems []itineraries.ItineraryItem
		var days []itineraries.Day
		iter := client.Collection("itineraries").Doc(itineraryID).Collection("days").OrderBy("day", firestore.Asc).Documents(ctx)
		for {
			dayDoc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				response.WriteErrorResponse(w, err)
				break
			}
			var day itineraries.Day
			dayDoc.DataTo(&day)
			days = append(days, day)
		}
		if daysCount < len(days) {
			daysRemaining := days[0:daysCount]
			numRemoved := len(days) - len(daysRemaining)
			daysRemoved := days[len(days) - numRemoved:]
			itineraryItemsChannel := make(chan []itineraries.ItineraryItem)
			var itineraryItemsData []itineraries.ItineraryItem
			for i := 0; i < len(daysRemoved); i++ {
				id := daysRemoved[i].ID
				go func(id string){
					itemsItr := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(id).Collection("itinerary_items").Documents(ctx)
					for {
						doc, err := itemsItr.Next()
						if err == iterator.Done {
							break
						}
						if err != nil {
							errorChannel <- err
							break
						}
						var itineraryItem itineraries.ItineraryItem
						doc.DataTo(&itineraryItem)
						itineraryItemsData = append(itineraryItemsData, itineraryItem)
						_, errItems := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(id).Collection("itinerary_items").Doc(itineraryItem.ID).Delete(ctx)
						if errItems != nil {
							errorChannel <- errItems
							break
						}

					}

					_, errDays := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(id).Delete(ctx)
					if errDays != nil {
						errorChannel <- errDays
					}
					itineraryItemsChannel <- itineraryItemsData
				}(id)

			}

			for i := 0; i < 1; i++ {
				select{
				case res := <- itineraryItemsChannel:
					itinerariesItems = res
				case err := <- errorChannel:
					response.WriteErrorResponse(w, err)
					return
				}
			}
					
			if len(itinerariesItems) > 0 {
				id := daysRemaining[len(daysRemaining)-1].ID
				for i:=0; i < len(itinerariesItems); i++ {
					_, errAdd := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(id).Collection("itinerary_items").Doc(itinerariesItems[i].ID).Set(ctx,itinerariesItems[i])
					if errAdd != nil {
						fmt.Println(errAdd)
						//response.WriteErrorResponse(w, errAdd)
						break
					}
				}
			}

		} else if daysCount > len(days) {
			lastIndex := len(days)
			daysAdded := daysCount - len(days)
			for i:=0; i < daysAdded; i++ {
				go func(lastIndex int, i int){
					daydoc, _, errCreate := client.Collection("itineraries").Doc(itineraryID).Collection("days").Add(ctx, map[string]interface{}{
						"day": lastIndex + i,
					})
					if errCreate != nil {
						// Handle any errors in an appropriate way, such as returning them.
						fmt.Println(errCreate)
						errorChannel <- errCreate
					}
		
					_, errCrUp := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(daydoc.ID).Set(ctx, map[string]interface{}{
						"id": daydoc.ID,
					},firestore.MergeAll)
					if errCrUp != nil {
						// Handle any errors in an appropriate way, such as returning them.
						fmt.Println(errCrUp)
						errorChannel <- errCrUp
					}
					dayIdsChannel <- daydoc.ID
				}(lastIndex, i)
			}
			var ids []string
			for i:=0; i < daysAdded; i++ {
				select{
					case res := <- dayIdsChannel:
						ids = append(ids, res)
					case err := <- errorChannel:
						response.WriteErrorResponse(w, err)
						return
				}
			}
		}
		
	}

	destinationData := map[string]interface{}{
		"success": true,
	}

	response.Write(w, destinationData, http.StatusOK)	
	return
}


// AddDestination function 
func AddDestination(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	exists := false
	decoder := json.NewDecoder(r.Body)
	destinationChannel := make(chan firestore.DocumentRef)
	var destination triptypes.Destination
	err := decoder.Decode(&destination)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}


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

	//Check Destination
	iter := client.Collection("trips").Doc(tripID).Collection("destinations").Where("destination_id", "==", destination.DestinationID).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			response.WriteErrorResponse(w, err)
			break
		}
		if doc.Exists() == true {
			exists = true;
			break;
		}
	}

	//Adding destinations
	if exists == true {
		response.Write(w, map[string]interface{}{
			"exists": true,
			"message": "This destination already exists for this trip",
		}, http.StatusConflict)	
		return
	}	
	go func(tripID string, destination triptypes.Destination){
		destDoc, _, errCreate := client.Collection("trips").Doc(tripID).Collection("destinations").Add(ctx, destination)
		if errCreate != nil {
			// Handle any errors in an appropriate way, such as returning them.
			fmt.Println(errCreate)
			response.WriteErrorResponse(w, errCreate)
		}

		_, err2 := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destDoc.ID).Set(ctx, map[string]interface{}{
			"id": destDoc.ID,
		},firestore.MergeAll)
		if err2 != nil {
			// Handle any errors in an appropriate way, such as returning them.
			response.WriteErrorResponse(w, err2)
		}
		destinationChannel <- *destDoc
	}(tripID, destination)

	var dest firestore.DocumentRef
	for i:=0; i < 1; i++ {
		select{
		case res := <- destinationChannel:
			dest = res
		}
	}
	
	
	
	destinationData := map[string]interface{}{
		"destination": dest,
	}

	response.Write(w, destinationData, http.StatusOK)	
	return
}

// DeleteDestination function 
func DeleteDestination(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	destinationID := mux.Vars(r)["destinationId"]

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
	
	_, errDelete := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Delete(ctx)
	if errDelete != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errDelete)
		response.WriteErrorResponse(w, err)
		return
	}
	
	deleteData := map[string]interface{}{
		"success": true,
	}

	response.Write(w, deleteData, http.StatusOK)	
	return
}

// DeleteTrip function 
func DeleteTrip(w http.ResponseWriter, r *http.Request) {
	tripID := mux.Vars(r)["tripId"]
	var errorChannel = make(chan error)
	var destDeleteChannel = make(chan interface{})

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

	var dest []triptypes.Destination
	iter := client.Collection("trips").Doc(tripID).Collection("destinations").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			response.WriteErrorResponse(w, err)
			return
		}
		var destination triptypes.Destination
		doc.DataTo(&destination)
		dest = append(dest, destination)
	}

	for i:=0; i < len(dest); i++ {
		go func(tripID string, destination triptypes.Destination) {
			deleteRes, errDelete := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destination.ID).Delete(ctx)
			if errDelete != nil {
				// Handle any errors in an appropriate way, such as returning them.
				fmt.Println(errDelete)
				errorChannel <- errDelete
			}
			destDeleteChannel <- deleteRes
		}(tripID,dest[i])
	}

	count := 0

	_, errDelete := client.Collection("trips").Doc(tripID).Delete(ctx)
	if errDelete != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errDelete)
		response.WriteErrorResponse(w, err)
		return
	}

	for i:=0; i < len(dest); i++ {
		select{
		case <- destDeleteChannel:
			count = count + 1
		case err := <- errorChannel:
			response.WriteErrorResponse(w, err)
			return
		}
	}

	
	
	
	deleteData := map[string]interface{}{
		"destinations_deleted": count,
		"success": true,
	}

	response.Write(w, deleteData, http.StatusOK)	
	return
}
