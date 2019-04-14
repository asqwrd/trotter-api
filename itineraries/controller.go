package itineraries

import (
	//"encoding/json" //"sort"
	"encoding/json"
	"fmt"
	"time"
	"net/http" //"net/url"
	"github.com/asqwrd/trotter-api/triposo"
	firebase "firebase.google.com/go" 
	"github.com/asqwrd/trotter-api/places"
	"cloud.google.com/go/firestore"
	"github.com/asqwrd/trotter-api/response" 
	"github.com/asqwrd/trotter-api/utils" 
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"googlemaps.github.io/maps"
	"net/url"
	"strconv"

)

func collectionHandler(iter *firestore.DocumentIterator, client *firestore.Client) (map[string]interface{}, error){
	ctx := context.Background()
	var itineraries = make([]Itinerary,0)
	daysChannel := make(chan DaysChannel)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var itinerary Itinerary
		doc.DataTo(&itinerary)
		itineraries = append(itineraries, itinerary)
	}

	for i := 0; i < len(itineraries); i++ {
		go func(index int) {
			var days []Day
			iter := client.Collection("itineraries").Doc(itineraries[index].ID).Collection("days").OrderBy("day",firestore.Asc).Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				var day Day
				var itineraryItem ItineraryItem
				var itineraryItems = make([]ItineraryItem,0)
				doc.DataTo(&day)
				iterItems := doc.Ref.Collection("itinerary_items").Documents(ctx)
				for {
					i10ItemsDoc, err := iterItems.Next()
					if err == iterator.Done {
						break
					}
					i10ItemsDoc.DataTo(&itineraryItem)
					if itineraryItem.Poi != nil && len(itineraryItem.Poi.Images) > 0 {
						itineraryItem.Image = itineraryItem.Poi.Images[0].Sizes.Medium.Url
					}
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
				return nil, res.Error
			}
			itineraries[res.Index].Days = res.Days
		}
	}
	



	return map[string]interface{}{
		"itineraries": itineraries,
	}, nil
}
// GetItineraries function
func GetItineraries(w http.ResponseWriter, r *http.Request) {

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	//colorChannel := make(chan places.ColorChannel)
	var q *url.Values
	args := r.URL.Query()
	q = &args
	


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

	itinerariesCollection := client.Collection("itineraries").Limit(10)
	var queries firestore.Query
	var itr *firestore.DocumentIterator
	var public bool
	result, errPublic := strconv.ParseBool(q.Get("public"))
	if errPublic != nil {
		public = true
	}
	public = result
	if len(q.Get("public")) == 0 {
		queries = itinerariesCollection.Where("public", "==", true)	
	} else {
		queries = itinerariesCollection.Where("public", "==", public)	
	}

	if len(q.Get("destination")) > 0 {
		notNil := utils.CheckFirestoreQueryResults(ctx, queries)
		if notNil ==  true {
			queries = queries.Where("destination", "==", q.Get("destination"))
		} else {
			queries = itinerariesCollection.Where("destination", "==", q.Get("destination"))
		}
	}
	if len(q.Get("owner_id")) > 0 {
		queries = queries.Where("owner_id", "==", q.Get("owner_id"))

	} else {
		queries = queries.Where("owner_id", "==", "")
	}

	notNil := utils.CheckFirestoreQueryResults(ctx, queries)

	if notNil == true {
		itr = queries.Documents(ctx)
	} else {
		itr = itinerariesCollection.Documents(ctx)
	}

	tripsData,errData := collectionHandler(itr,client)
	if errData != nil {
		response.WriteErrorResponse(w, errData)
	}

	fmt.Print("Got Itineraries\n")

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
	iter := client.Collection("itineraries").Doc(itineraryID).Collection("days").OrderBy("day", firestore.Asc).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		var day Day
		var itineraryItems = make([]ItineraryItem,0)
		doc.DataTo(&day)
		iterItems := doc.Ref.Collection("itinerary_items").Documents(ctx)
		for {
			i10ItemsDoc, err := iterItems.Next()
			if err == iterator.Done {
				break
			}
			var itineraryItem ItineraryItem
			i10ItemsDoc.DataTo(&itineraryItem)
			if itineraryItem.Poi != nil && len(itineraryItem.Poi.Images) > 0 {
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

func optimizeItinerary(itineraryItems []ItineraryItem, matrix maps.DistanceMatrixResponse) ([]ItineraryItem){
	var rows = matrix.Rows
	var slice []map[string]interface{}
	var visited = make(map[string]interface{})
	
	for i := 0; i < len(itineraryItems); i++ {

		var colSlice []map[string]interface{}
		for j := 0; j < len(rows[i].Elements); j++ {
			colSlice = append(colSlice, map[string]interface{}{
				"element": rows[i].Elements[j],
				"item" : itineraryItems[j],
			})
		}
		slice = append(slice, map[string]interface{}{
			"columns": colSlice,
			"item" : itineraryItems[i],
			"index": i,
		})
		
	}
	var queue []map[string]interface{}
	queue = append(queue,slice[0])
	visited[slice[0]["item"].(ItineraryItem).ID] = true
	var output []ItineraryItem
	for (len(queue) > 0 ) {
		var read = queue[0]
		queue = queue[1:]
		output = append(output,read["item"].(ItineraryItem))
		var min int
		var next map[string]interface{}
		var nextID string
		var elements = read["columns"].([]map[string]interface{})
		var travel *maps.DistanceMatrixElement

		for k := 0; k < len(elements); k++ {
			var columnDistance = elements[k]["element"].(*maps.DistanceMatrixElement).Distance.Meters
			var col = elements[k]["item"].(ItineraryItem)
			if (min == 0  || min > columnDistance) && visited[col.ID] == nil {
				min = columnDistance
				nextID = col.ID
				travel = elements[k]["element"].(*maps.DistanceMatrixElement)
			}
		}
		if visited[nextID] == nil {
			for i:=0; i < len(slice); i++ {
				if slice[i]["item"].(ItineraryItem).ID == nextID {
					var item = slice[i]["item"].(ItineraryItem)
					item.Travel = *travel
					next = slice[i]
					next["item"] = item
				
					visited[nextID] = true
					queue = append(queue,next)
					break;
				}
			}
		}
		
	}
	return output

}

func getDay(w http.ResponseWriter, r *http.Request, justAdded *string, optimize bool){
	itineraryID := mux.Vars(r)["itineraryId"]
	dayID := mux.Vars(r)["dayId"]

	errorChannel := make(chan error)
	matrixChannel := make(chan maps.DistanceMatrixResponse)

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


	itinerary, err := getItinerary(itineraryID)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	snap, err := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Get(ctx)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}
	day := Day{ItineraryItems: []ItineraryItem{}}
	snap.DataTo(&day)
	day.ItineraryItems = make([]ItineraryItem,0)
	var itineraryItems = make([]ItineraryItem,0)
	docs := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Documents(ctx)
	for{
		i10ItemDocs, err := docs.Next()
			if err == iterator.Done {
				break
			}
		var itineraryItem ItineraryItem
		i10ItemDocs.DataTo(&itineraryItem)
		itineraryItems = append(itineraryItems, itineraryItem)
	}

	googleClient, err := places.InitGoogle()
	if err != nil  {
		response.WriteErrorResponse(w, err)
	}

	if optimize == false {
		go func(itinerary interface{}) {
			var locations []string
			locations = append(locations,fmt.Sprintf("%g,%g",itinerary.(Itinerary).Location.Latitude , itinerary.(Itinerary).Location.Longitude))
			for i:=0; i < len(itineraryItems); i++ {
				location := fmt.Sprintf("%g,%g", itineraryItems[i].Poi.Location.Lat,itineraryItems[i].Poi.Location.Lng)
				if(itineraryItems[i].Poi.Coordinates != nil){
					location = fmt.Sprintf("%g,%g", itineraryItems[i].Poi.Coordinates.Latitude,itineraryItems[i].Poi.Coordinates.Longitude)
				}
				locations = append(locations,location);
				if itineraryItems[i].Poi != nil && len(itineraryItems[i].Poi.Images) > 0 {
					itineraryItems[i].Image = itineraryItems[i].Poi.Images[0].Sizes.Medium.Url
				

					colors, err := places.GetColor(itineraryItems[i].Image)
					if err != nil {
						errorChannel <- err
						return 
					}

					if len(colors.Vibrant) > 0 {
						itineraryItems[i].Color = colors.Vibrant
					} else if len(colors.Muted) > 0 {
						itineraryItems[i].Color = colors.Muted
					} else if len(colors.LightVibrant) > 0 {
						itineraryItems[i].Color = colors.LightVibrant
					} else if len(colors.LightMuted) > 0 {
						itineraryItems[i].Color = colors.LightMuted
					} else if len(colors.DarkVibrant) > 0 {
						itineraryItems[i].Color = colors.DarkVibrant 
					} else if len(colors.DarkMuted) > 0 {
						itineraryItems[i].Color = colors.DarkMuted
					}
				}
			}
			r := &maps.DistanceMatrixRequest{
				Origins:      locations,
				Destinations: locations,
			}
			matrix,err := googleClient.DistanceMatrix(ctx,r)
			if err != nil {
				errorChannel <- err
			}

			matrixChannel <- *matrix
			
			
		}(itinerary["itinerary"])

		for i:=0; i < 1; i++ {
			select{
			case matrix := <- matrixChannel:
				var head ItineraryItem
				itineraryItems = append([]ItineraryItem{head}, itineraryItems...)
				day.ItineraryItems = optimizeItinerary(itineraryItems,matrix)
			case err := <- errorChannel:
				response.WriteErrorResponse(w, err)
				return
			}
		}
	} else {
		for i:=0; i < len(itineraryItems); i++ {
			if itineraryItems[i].Poi != nil && len(itineraryItems[i].Poi.Images) > 0 {
				itineraryItems[i].Image = itineraryItems[i].Poi.Images[0].Sizes.Medium.Url
			

				colors, err := places.GetColor(itineraryItems[i].Image)
				if err != nil {
					errorChannel <- err
					return 
				}

				if len(colors.Vibrant) > 0 {
					itineraryItems[i].Color = colors.Vibrant
				} else if len(colors.Muted) > 0 {
					itineraryItems[i].Color = colors.Muted
				} else if len(colors.LightVibrant) > 0 {
					itineraryItems[i].Color = colors.LightVibrant
				} else if len(colors.LightMuted) > 0 {
					itineraryItems[i].Color = colors.LightMuted
				} else if len(colors.DarkVibrant) > 0 {
					itineraryItems[i].Color = colors.DarkVibrant 
				} else if len(colors.DarkMuted) > 0 {
					itineraryItems[i].Color = colors.DarkMuted
				}
			}
		}
		day.ItineraryItems = itineraryItems
	}

	dayData := map[string]interface{}{
		"day": day,
		"itinerary": itinerary,
		"justAdded": justAdded,
	}

	response.Write(w, dayData, http.StatusOK)
	return
}

//GetDay func
func GetDay(w http.ResponseWriter, r *http.Request) {

	getDay(w,r,nil, false)
	return

}

// CreateItineraryHelper function
func CreateItineraryHelper(tripID string, destinationID string, itinerary Itinerary) (map[string]interface{}, error){
	dayChannel := make(chan string)
	errorChannel := make(chan error)
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

	doc, _, errCreate := client.Collection("itineraries").Add(ctx, itinerary)
	if errCreate != nil {
		// Handle any errors in an appropriate way, such as returning them.
		return nil, errCreate
	}

	_, err2 := client.Collection("itineraries").Doc(doc.ID).Set(ctx, map[string]interface{}{
		"id": doc.ID,
		"public": false,
	},firestore.MergeAll)
	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		return nil, err2
	}

	_, errTrip := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Set(ctx, map[string]interface{}{
		"itinerary_id": doc.ID,
	},firestore.MergeAll) 
	if errTrip != nil {
		// Handle any errors in an appropriate way, such as returning them.
		return nil, errTrip
	}

	//Adding days
	var daysCount = 0
	
	endtm := time.Unix(itinerary.EndDate, 0)
	starttm := time.Unix(itinerary.StartDate, 0)

	diff := endtm.Sub(starttm)
	daysCount = int(diff.Hours()/24) + 1 //include first day


	for i:=0; i < daysCount; i++ {
		go func(index int, itineraryID string){
			daydoc, _, errCreate := client.Collection("itineraries").Doc(itineraryID).Collection("days").Add(ctx, map[string]interface{}{
				"day": index,
			})
			if errCreate != nil {
				// Handle any errors in an appropriate way, such as returning them.
				errorChannel <- errCreate
			}

			_, errCrUp := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(daydoc.ID).Set(ctx, map[string]interface{}{
				"id": daydoc.ID,
			},firestore.MergeAll)
			if errCrUp != nil {
				// Handle any errors in an appropriate way, such as returning them.
				errorChannel <- errCrUp
			}

			dayChannel <- doc.ID
		}(i, doc.ID)
	}
	var dayIDS []string
	for i:=0; i < daysCount; i++ {
		select{
		case res := <- dayChannel:
			dayIDS = append(dayIDS,res)
		case err := <-errorChannel:
			return nil, err
		}
	}
	
	id := doc.ID
	return map[string]interface{}{
		"id": id,
	}, nil
}
//CreateItinerary func
func CreateItinerary(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var itinerary ItineraryRes
	err := decoder.Decode(&itinerary)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	itineraryData, err := CreateItineraryHelper(itinerary.Itinerary.TripID, itinerary.TripDestinationID, itinerary.Itinerary)
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}
	response.Write(w, itineraryData, http.StatusOK)
	return
}

//AddToDay func
func AddToDay(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]
	dayID := mux.Vars(r)["dayId"]
	decoder := json.NewDecoder(r.Body)
	var itineraryItem ItineraryItem
	q := r.URL.Query()
	err := decoder.Decode(&itineraryItem)
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

	

	doc, _, err2 := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Add(ctx, itineraryItem)
	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		response.WriteErrorResponse(w, err2)
		return
	}

	_, errSet := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(doc.ID).Set(ctx, map[string]interface{}{
		"id": doc.ID,
	},firestore.MergeAll)
	if err != nil {
		response.WriteErrorResponse(w, errSet)
		return
	}
	
	id := &doc.ID
	if q.Get("optimize") == "true" {
		print("optimize \n")
		getDay(w,r,id,true)
	} else {
		print("full \n")
		getDay(w,r,id,false)
	}
	
	fmt.Println("added")
	return 


}

// DeleteItineraryItem function 
func DeleteItineraryItem(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]
	dayID := mux.Vars(r)["dayId"]
	place := mux.Vars(r)["placeId"]

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
	
	_, errDelete := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(place).Delete(ctx)
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