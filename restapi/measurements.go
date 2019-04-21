package restapi

import (
	"net/http"
	// "github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/davecgh/go-spew/spew"
	"fmt"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"


)

// This is dumb, but I guess it will work for now. I am clearly not understanding something.
type MeasurementHandler struct {}
func (h MeasurementHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	InsertMeasurement(w, r)
}

func InsertMeasurement(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the HomePage!")
	fmt.Println("Endpoint Hit: homePage")
	// fmt.Printf("%s", spew.Sdump(r.Context()))

	user, _ := authMiddleware.UserFromContext(r.Context())
	fmt.Printf("%s", spew.Sdump(user))
	// if user == nil {
	// 	return nil, ErrUserNotAuthenticated
	// }

}
