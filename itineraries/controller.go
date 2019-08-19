package itineraries

import (
	//"encoding/json" //"sort"
	"encoding/json"
	"fmt"
	"net/http" //"net/url"
	"net/url"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/places"
	"github.com/asqwrd/trotter-api/response"
	"github.com/asqwrd/trotter-api/triposo"
	"github.com/asqwrd/trotter-api/types"
	"github.com/asqwrd/trotter-api/utils"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"googlemaps.github.io/maps"
	"gopkg.in/maddevsio/fcm.v1"
)

func collectionHandler(iter *firestore.DocumentIterator, client *firestore.Client) (map[string]interface{}, error) {
	ctx := context.Background()
	var itineraries = make([]Itinerary, 0)
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
			iter := client.Collection("itineraries").Doc(itineraries[index].ID).Collection("days").OrderBy("day", firestore.Asc).Documents(ctx)
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				var day Day
				var itineraryItem ItineraryItem
				var itineraryItems = make([]ItineraryItem, 0)
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
					itineraryItems = append(itineraryItems, itineraryItem)

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

	totalDoc, errTotal := client.Collection("itineraries").Doc("total_public").Get(ctx)
	if errTotal != nil {
		return nil, errTotal
	}

	total := totalDoc.Data()

	return map[string]interface{}{
		"itineraries":  itineraries,
		"total_public": total,
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
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	var itinerariesCollection firestore.Query
	if len(q.Get("last")) > 0 {
		itinerariesCollection = client.Collection("itineraries").OrderBy("id", firestore.Asc).StartAfter(q.Get("last")).Limit(10)
	} else {
		itinerariesCollection = client.Collection("itineraries").OrderBy("id", firestore.Asc).Limit(10)
	}
	var queries firestore.Query
	var itr *firestore.DocumentIterator
	var public bool

	if len(q.Get("public")) > 0 {
		result, errPublic := strconv.ParseBool(q.Get("public"))
		if errPublic != nil {
			public = true
		}
		public = result
		queries = itinerariesCollection.Where("public", "==", public)
	}

	if len(q.Get("destination")) > 0 {
		notNil := utils.CheckFirestoreQueryResults(ctx, queries)
		if notNil == true {
			queries = queries.Where("destination", "==", q.Get("destination"))
		} else {
			queries = itinerariesCollection.Where("destination", "==", q.Get("destination"))
		}
	}
	if len(q.Get("user_id")) > 0 {
		queries = queries.Where("travelers", "array-contains", q.Get("user_id"))

	} else {
		queries = queries.Where("owner_id", "==", "")
	}

	notNil := utils.CheckFirestoreQueryResults(ctx, queries)

	if notNil == true {
		itr = queries.Documents(ctx)
	} else {
		itr = itinerariesCollection.Documents(ctx)
	}

	itineraryData, errData := collectionHandler(itr, client)
	if errData != nil {
		fmt.Println(errData)
		response.WriteErrorResponse(w, errData)
		return
	}

	fmt.Println("Got Itineraries")

	response.Write(w, itineraryData, http.StatusOK)
	return
}

//Private getItinerary funtion
func getItinerary(itineraryID string) (map[string]interface{}, error) {
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	errorChannel := make(chan error)
	destinationChannel := make(chan map[string]interface{})
	app, err := firebase.NewApp(ctx, nil, sa)
	fmt.Println("Got Itinerary")
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

	go func(id string) {
		parent, err := triposo.GetLocation(id)
		if err != nil {
			errorChannel <- err
		}
		parentParam := *parent
		var destination triposo.InternalPlace
		var colors *places.Colors
		if len(parentParam) > 0 {
			destination = places.FromTriposoPlace(parentParam[0], parentParam[0].Type)
			colorsRes, err := places.GetColor(destination.Image)
			if err != nil {
				errorChannel <- err
				return
			}
			colors = colorsRes
		}

		destinationChannel <- map[string]interface{}{
			"colors":      colors,
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
		var itineraryItems = make([]ItineraryItem, 0)
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
			itineraryItems = append(itineraryItems, itineraryItem)

		}
		day.ItineraryItems = itineraryItems
		days = append(days, day)
	}
	itinerary.Days = days

	var destination triposo.InternalPlace
	var color string
	for i := 0; i < 1; i++ {
		select {
		case res := <-destinationChannel:
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
		case err := <-errorChannel:
			return nil, err
		}
	}
	hotels := []map[string]interface{}{}
	if len(itinerary.TripID) > 0 {
		fmt.Println(destination.ID)
		destItr := client.Collection("trips").Doc(itinerary.TripID).Collection("destinations").Where("destination_id", "==", destination.ID).Documents(ctx)
		for {
			destDoc, errDest := destItr.Next()
			if errDest == iterator.Done {
				break
			}
			if errDest != nil {
				return nil, errDest
			}

			detailsIter := client.Collection("trips").Doc(itinerary.TripID).Collection("destinations").Doc(destDoc.Ref.ID).Collection("flights_accomodations").Documents(ctx)
			for {
				detailsDoc, errDetail := detailsIter.Next()
				if errDetail == iterator.Done {
					break
				}
				if errDetail != nil {
					return nil, errDetail
				}
				var flightAccomodation types.FlightsAndAccomodations
				detailsDoc.DataTo(&flightAccomodation)
				for _, segment := range flightAccomodation.Segments {
					if segment.Type == "Hotel" {
						travelers := []types.User{}
						for _, id := range flightAccomodation.Travelers {
							userDoc, errUser := client.Collection("users").Doc(id).Get(ctx)
							if errUser != nil {
								return nil, errUser
							}
							var user types.User
							userDoc.DataTo(&user)
							travelers = append(travelers, user)

						}
						detail := map[string]interface{}{
							"hotel":     segment,
							"travelers": travelers,
						}
						hotels = append(hotels, detail)
					}
				}

			}

		}

	}

	itineraryData := map[string]interface{}{
		"itinerary":   itinerary,
		"destination": destination,
		"hotels":      hotels,
		"color":       color,
	}
	return itineraryData, err
}

//GetItinerary function
func GetItinerary(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]
	//tripChannel := make(chan Trip)
	itineraryData, err := getItinerary(itineraryID)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	response.Write(w, itineraryData, http.StatusOK)
	return
}

//ChangeStartLocation function
func ChangeStartLocation(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	decoder := json.NewDecoder(r.Body)
	var location StartLocation
	err := decoder.Decode(&location)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	app, errApp := firebase.NewApp(ctx, nil, sa)
	fmt.Println("Change start location")
	if errApp != nil {
		fmt.Println(errApp)
		response.WriteErrorResponse(w, errApp)
		return
	}

	client, errClient := app.Firestore(ctx)
	if errClient != nil {
		fmt.Println(errClient)
		response.WriteErrorResponse(w, errClient)
		return
	}

	defer client.Close()
	_, errUpdate := client.Collection("itineraries").Doc(itineraryID).Set(ctx, map[string]interface{}{
		"start_location": location,
	}, firestore.MergeAll)
	if errUpdate != nil {
		fmt.Println(errUpdate)
		response.WriteErrorResponse(w, errUpdate)
		return
	}

	response.Write(w, map[string]interface{}{
		"start_location": location,
		"success":        true,
	}, http.StatusOK)
	return
}

func optimizeItinerary(itineraryItems []ItineraryItem, matrix maps.DistanceMatrixResponse) []ItineraryItem {
	var rows = matrix.Rows
	var slice []map[string]interface{}
	var visited = make(map[string]interface{})

	for i := 0; i < len(itineraryItems); i++ {

		var colSlice []map[string]interface{}
		for j := 0; j < len(rows[i].Elements); j++ {
			colSlice = append(colSlice, map[string]interface{}{
				"element": rows[i].Elements[j],
				"item":    itineraryItems[j],
			})
		}
		slice = append(slice, map[string]interface{}{
			"columns": colSlice,
			"item":    itineraryItems[i],
			"index":   i,
		})

	}
	var queue []map[string]interface{}
	queue = append(queue, slice[0])
	visited[slice[0]["item"].(ItineraryItem).ID] = true
	var output []ItineraryItem
	for len(queue) > 0 {
		var read = queue[0]
		queue = queue[1:]
		output = append(output, read["item"].(ItineraryItem))
		var min int
		var next map[string]interface{}
		var nextID string
		var elements = read["columns"].([]map[string]interface{})
		var travel *maps.DistanceMatrixElement

		for k := 0; k < len(elements); k++ {
			var columnDistance = elements[k]["element"].(*maps.DistanceMatrixElement).Distance.Meters
			var col = elements[k]["item"].(ItineraryItem)
			if (min == 0 || min > columnDistance) && visited[col.ID] == nil {
				min = columnDistance
				nextID = col.ID
				travel = elements[k]["element"].(*maps.DistanceMatrixElement)
			}
		}
		if visited[nextID] == nil {
			for i := 0; i < len(slice); i++ {
				if slice[i]["item"].(ItineraryItem).ID == nextID {
					var item = slice[i]["item"].(ItineraryItem)
					item.Travel = *travel
					next = slice[i]
					next["item"] = item

					visited[nextID] = true
					queue = append(queue, next)
					break
				}
			}
		}

	}
	return output

}

func getDay(w http.ResponseWriter, r *http.Request, justAdded *string, optimize bool) {
	itineraryID := mux.Vars(r)["itineraryId"]
	dayID := mux.Vars(r)["dayId"]

	errorChannel := make(chan error)
	matrixChannel := make(chan maps.DistanceMatrixResponse)
	var q *url.Values
	args := r.URL.Query()
	q = &args
	latlng := q.Get("latlng")

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	itinerary, err := getItinerary(itineraryID)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	snap, err := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Get(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}
	day := Day{ItineraryItems: []ItineraryItem{}}
	snap.DataTo(&day)
	day.ItineraryItems = make([]ItineraryItem, 0)
	var itineraryItems = make([]ItineraryItem, 0)
	docs := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Documents(ctx)
	for {
		i10ItemDocs, err := docs.Next()
		if err == iterator.Done {
			break
		}
		var itineraryItem ItineraryItem
		i10ItemDocs.DataTo(&itineraryItem)
		itineraryItems = append(itineraryItems, itineraryItem)
	}

	googleClient, err := places.InitGoogle()
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
	}

	if optimize == false {
		go func(itinerary interface{}) {
			//fmt.Println(itinerary)
			var locations []string
			var chunks [][]string
			start := itinerary.(Itinerary).Location
			if itinerary.(Itinerary).StartLocation != nil {
				start = itinerary.(Itinerary).StartLocation.Location
			}

			if len(latlng) > 0 {
				locations = append(locations, latlng)
			} else {
				locations = append(locations, fmt.Sprintf("%g,%g", start.Latitude, start.Longitude))
			}

			for i := 0; i < len(itineraryItems); i++ {
				location := fmt.Sprintf("%g,%g", itineraryItems[i].Poi.Location.Lat, itineraryItems[i].Poi.Location.Lng)
				if itineraryItems[i].Poi.Coordinates != nil {
					location = fmt.Sprintf("%g,%g", itineraryItems[i].Poi.Coordinates.Latitude, itineraryItems[i].Poi.Coordinates.Longitude)
				}
				if itineraryItems[i].AddedBy != nil {
					userDoc, errUser := client.Collection("users").Doc(*itineraryItems[i].AddedBy).Get(ctx)
					if errUser != nil {
						errorChannel <- errUser
						return
					}
					var user types.User
					userDoc.DataTo(&user)
					itineraryItems[i].AddedByFull = &user
				}
				locations = append(locations, location)
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

			batchSize := 10

			for batchSize < len(locations) {
				locations, chunks = locations[batchSize:], append(chunks, locations[0:batchSize:batchSize])
			}
			chunks = append(chunks, locations)

			var matrix *maps.DistanceMatrixResponse
			for i := 0; i < len(chunks); i++ {

				r := &maps.DistanceMatrixRequest{
					Origins:      chunks[i],
					Destinations: chunks[i],
				}
				res, err := googleClient.DistanceMatrix(ctx, r)
				if err != nil {
					fmt.Println(err)
					errorChannel <- err
					return
				}
				if matrix == nil {
					matrix = res
				} else {
					matrix.Rows = append(matrix.Rows, res.Rows...)
				}

			}
			matrixChannel <- *matrix

		}(itinerary["itinerary"])

		for i := 0; i < 1; i++ {
			select {
			case matrix := <-matrixChannel:
				var head ItineraryItem
				itineraryItems = append([]ItineraryItem{head}, itineraryItems...)
				day.ItineraryItems = optimizeItinerary(itineraryItems, matrix)
			case err := <-errorChannel:
				fmt.Println(err)
				response.WriteErrorResponse(w, err)
				return
			}
		}
	} else {
		for i := 0; i < len(itineraryItems); i++ {
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
		"day":       day,
		"itinerary": itinerary,
		"justAdded": justAdded,
	}

	response.Write(w, dayData, http.StatusOK)
	return
}

//GetDay func
func GetDay(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got day")
	getDay(w, r, nil, false)
	return

}

// CreateItineraryHelper function
func CreateItineraryHelper(tripID string, destinationID string, itinerary Itinerary) (map[string]interface{}, error) {
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
		"id":     doc.ID,
		"public": false,
	}, firestore.MergeAll)
	if err2 != nil {
		// Handle any errors in an appropriate way, such as returning them.
		return nil, err2
	}

	_, errTrip := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Set(ctx, map[string]interface{}{
		"itinerary_id": doc.ID,
	}, firestore.MergeAll)
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

	for i := 0; i < daysCount; i++ {
		go func(index int, itineraryID string) {
			daydoc, _, errCreate := client.Collection("itineraries").Doc(itineraryID).Collection("days").Add(ctx, map[string]interface{}{
				"day": index,
			})
			if errCreate != nil {
				// Handle any errors in an appropriate way, such as returning them.
				errorChannel <- errCreate
				return
			}

			_, errCrUp := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(daydoc.ID).Set(ctx, map[string]interface{}{
				"id": daydoc.ID,
			}, firestore.MergeAll)
			if errCrUp != nil {
				// Handle any errors in an appropriate way, such as returning them.
				errorChannel <- errCrUp
				return
			}

			dayChannel <- doc.ID
		}(i, doc.ID)
	}
	var dayIDS []string
	for i := 0; i < daysCount; i++ {
		select {
		case res := <-dayChannel:
			dayIDS = append(dayIDS, res)
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
		fmt.Println(err)
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
	var id *string
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
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	iter := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Where("poi_id", "==", itineraryItem.PoiID).Documents(ctx)
	for {
		docCheck, errCheck := iter.Next()
		if errCheck == iterator.Done {
			break
		}
		if errCheck != nil {
			fmt.Println(errCheck)
			response.WriteErrorResponse(w, errCheck)
			return
		}
		id = &docCheck.Ref.ID
	}

	if id == nil {
		doc, _, err2 := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Add(ctx, itineraryItem)
		if err2 != nil {
			// Handle any errors in an appropriate way, such as returning them.
			fmt.Println(err2)
			response.WriteErrorResponse(w, err2)
			return
		}

		_, errSet := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(doc.ID).Set(ctx, map[string]interface{}{
			"id": doc.ID,
		}, firestore.MergeAll)
		if errSet != nil {
			fmt.Println(errSet)
			response.WriteErrorResponse(w, errSet)
			return
		}

		id = &doc.ID
	}

	if q.Get("optimize") == "true" {
		print("optimize \n")
		getDay(w, r, id, true)
	} else {
		print("full \n")
		getDay(w, r, id, false)
	}

	fmt.Println("added")
	return

}

// DeleteItineraryItem function
func DeleteItineraryItem(w http.ResponseWriter, r *http.Request) {
	itineraryID := mux.Vars(r)["itineraryId"]
	dayID := mux.Vars(r)["dayId"]
	place := mux.Vars(r)["placeId"]
	fmt.Println("Delete Itinerary Item")

	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	_, errDelete := client.Collection("itineraries").Doc(itineraryID).Collection("days").Doc(dayID).Collection("itinerary_items").Doc(place).Delete(ctx)
	if errDelete != nil {
		// Handle any errors in an appropriate way, such as returning them.
		fmt.Println(errDelete)
		response.WriteErrorResponse(w, errDelete)
		return
	}

	deleteData := map[string]interface{}{
		"success": true,
	}

	response.Write(w, deleteData, http.StatusOK)
	return
}

// TestNotification function
func TestNotification(w http.ResponseWriter, r *http.Request) {
	sa := option.WithCredentialsFile("serviceAccountKey.json")
	ctx := context.Background()
	c := fcm.NewFCM(types.SERVER_KEY)

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Println(err)
		response.WriteErrorResponse(w, err)
		return
	}

	defer client.Close()

	userDoc, errUser := client.Collection("users").Doc("BjPpGEpI0ERGoCnGSdalv22jbV73").Get(ctx)
	if errUser != nil {
		fmt.Println(errUser)
		response.WriteErrorResponse(w, errUser)
		return
	}
	var user types.User
	userDoc.DataTo(&user)
	iter := client.Collection("users").Doc("BjPpGEpI0ERGoCnGSdalv22jbV73").Collection("devices").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		}

		navigateData := map[string]interface{}{
			"itineraryId":   "uRrfmJintOxNVhevP74p",
			"dayId":         "5lOZSEJz345IJmOZEED7",
			"startLocation": map[string]interface{}{"lat": 3.143497, "lng": 101.704094},
			"level":         "itinerary/day/edit",
		}

		var token types.Token
		doc.DataTo(&token)
		data := map[string]interface{}{
			"focus":            "trips",
			"click_action":     "FLUTTER_NOTIFICATION_CLICK",
			"notificationData": navigateData,
		}

		notification, err := c.Send(fcm.Message{
			Data:             data,
			RegistrationIDs:  []string{token.Token},
			ContentAvailable: true,
			Priority:         fcm.PriorityHigh,
			Notification: fcm.Notification{
				Title:       "Hello",
				Body:        "World",
				ClickAction: "FLUTTER_NOTIFICATION_CLICK",
				//Badge: user.PhotoURL,
			},
		})
		if err != nil {
			fmt.Println(err)
			response.WriteErrorResponse(w, err)
			return
		}
		fmt.Println("Status Code   :", notification.StatusCode)
		fmt.Println("Success       :", notification.Success)
		fmt.Println("Fail          :", notification.Fail)
		fmt.Println("Canonical_ids :", notification.CanonicalIDs)
		fmt.Println("Topic MsgId   :", notification.MsgID)

	}

	response.Write(w, map[string]interface{}{
		"ok": true,
	}, http.StatusOK)

	return

}
