package main

import (
	"fmt"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"os"
	"flag"
)

func main() {
	// TODO: switch from flag to Cobra
	// Can I just use a database url here?  I think I can.
	dbName, _ := os.LookupEnv("DATABASE_NAME")
	dbUser, _ := os.LookupEnv("DATABASE_USER")
	dbPassword, _ := os.LookupEnv("DATABASE_PASSWORD")
	dbHost, _ := os.LookupEnv("DATABASE_HOST")
	dbPort, dbPortSet := os.LookupEnv("DATABASE_PORT")
	if !dbPortSet {
		dbPort = "5432" // again, assume postgres
	}
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, _ := sqlx.Connect("postgres", connectionString)

	// subcommands
	clearTokenCmd := flag.NewFlagSet("clear", flag.ExitOnError)
	makeTokenCmd := flag.NewFlagSet("maketoken", flag.ExitOnError)
	//tokenId := clearTokenCmd.String("tokenId", "", "Token id to clear")
	email := makeTokenCmd.String("email", "", "Email address of user")
	// flag.Parse()

	if len(os.Args) == 1 {
		fmt.Println("usage: wortuser <command> [<args>]")
		fmt.Println("The most commonly used wortuser commands are: ")
		fmt.Println(" cleartoken   Find the most recent auth token for a user")
		fmt.Println(" maketoken  Make an authentication token for a user")
		return
	}

	switch os.Args[1] {
	case "cleartoken":
			clearTokenCmd.Parse(os.Args[2:])
		case "maketoken":
			makeTokenCmd.Parse(os.Args[2:])
		default:
			fmt.Printf("%q is not valid command.\n", os.Args[1])
			os.Exit(2)
	}

	if clearTokenCmd.Parsed() {
		fmt.Println("Clear Token")
	}

	if makeTokenCmd.Parsed() {
		fmt.Printf("Making token for user: %s\n", *email)
		token, err := makeToken(*email, db)
		if err != nil {
			fmt.Printf("Error creating token: %v\n", err)
		} else {
			fmt.Printf("Created token: %s\n", token.ForAuthenticationHeader())
		}

	}

}

// Make token for user...  should really take User
func makeToken(username string, db *sqlx.DB) (*worrywort.AuthToken, error) {
	user, err := worrywort.FindUser(map[string]interface{}{"email": username}, db)
	if err != nil {
		return nil, err
	}

	token, err := worrywort.GenerateTokenForUser(*user, worrywort.TOKEN_SCOPE_ALL)
	if err != nil {
		return nil, err
	}

	err = token.Save(db)
	if err != nil {
		return nil, err
	}
	return &token, nil
}
