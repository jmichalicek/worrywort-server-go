package restapi

//
// import (
// 	"net/http"
// 	"github.com/jmichalicek/worrywort-server-go/worrywort"
// )
//
//
// // I might move all of this to cmd/worrywortd and see if I can also have it wrap the graphql bits
// // This comes from a combo of https://ryanmccue.ca/how-to-create-restful-api-golang-standard-library/ and
// // https://github.com/nerdyc/testable-golang-web-service/blob/master/service.go
//
// type Server struct {
// 	// router *httprouter.Router
// 	db     *Database
// }
//
// func NewServer(db *Database) *Server {
// 	// router := httprouter.New()
// 	server := &Server{
// 		// router: router,
// 		db:     db,
// 	}
//
// 	// server.setupRoutes()
// 	return server
// }
