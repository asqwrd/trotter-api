package traxo

import (
	//"sort"

	"fmt"
	"net/http" //"net/url"

	//"github.com/asqwrd/trotter-api/triposo"
	"net/url"

	//	firebase "firebase.google.com/go"
	"github.com/asqwrd/trotter-api/response"
	//	"golang.org/x/net/context"
	//	"google.golang.org/api/option"
)

// GetConfirmations function
func GetConfirmations(w http.ResponseWriter, r *http.Request) {
	var q *url.Values
	args := r.URL.Query()
	q = &args
	confirmations := make(chan Confirmation)

	email := q.Get("email")
	tripID := q.Get("tripId")

	fmt.Println(email)
	fmt.Println(tripID)

	emails, err := GetEmails()
	if err != nil {
		response.WriteErrorResponse(w, err)
		return
	}

	filteredEmails := []Email{}
	for _, x := range *emails {
		if x.FromAddress == email {
			filteredEmails = append(filteredEmails, x)
		}
	}
	for _, confirmation := range filteredEmails {
		go func(confirmation Email) {
			res, errCon := GetEmail(confirmation.ID)
			if errCon != nil {
				response.WriteErrorResponse(w, errCon)
				return
			}

			fmt.Println(res.ID)

			confirmations <- *res

		}(confirmation)
	}

	results := []Confirmation{}
	for i := 0; i < len(*emails); i++ {
		select {
		case res := <-confirmations:
			results = append(results, res)
		}
	}

	fmt.Println(results)

	// sa := option.WithCredentialsFile("serviceAccountKey.json")
	// ctx := context.Background()

	// app, err := firebase.NewApp(ctx, nil, sa)
	// if err != nil {
	// 	response.WriteErrorResponse(w, err)
	// 	return
	// }

	// client, err := app.Firestore(ctx)
	// if err != nil {
	// 	response.WriteErrorResponse(w, err)
	// 	return
	// }

	// defer client.Close()

	// _, errDelete := client.Collection("trips").Doc(tripID).Collection("destinations").Doc(destinationID).Delete(ctx)
	// if errDelete != nil {
	// 	// Handle any errors in an appropriate way, such as returning them.
	// 	fmt.Println(errDelete)
	// 	response.WriteErrorResponse(w, err)
	// 	return
	// }

	// deleteData := map[string]interface{}{
	// 	"success": true,
	// }

	response.Write(w, map[string]interface{}{
		"results": results,
		"success": true,
	}, http.StatusOK)
	return
}
