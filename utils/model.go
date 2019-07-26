package utils

import (
	"cloud.google.com/go/firestore"
	"golang.org/x/net/context"
	"github.com/asqwrd/trotter-api/types"

)

// Block Type
type Block struct {
	Try     func()
	Catch   func(Exception)
	Finally func()
}

// Exception type
type Exception interface{}

// Throw function
func Throw(up Exception) {
	panic(up)
}

// Do function
func (tcf Block) Do() {
	if tcf.Finally != nil {

		defer tcf.Finally()
	}
	if tcf.Catch != nil {
		defer func() {
			if r := recover(); r != nil {
				tcf.Catch(r)
			}
		}()
	}
	tcf.Try()
}

// CheckFirestoreQueryResults function
func CheckFirestoreQueryResults(ctx context.Context, query firestore.Query) bool {

	defer func() bool {
		if err := recover(); err != nil {
			return false
		}
		return true
	}()
	query.Documents(ctx)

	return true
}

// FindInTripGroup function
func FindInTripGroup(group []interface{}, queryUser types.User) bool {
	for _, s := range group {
		if s == queryUser.UID {
			return true
		}
	}

	return false
}
