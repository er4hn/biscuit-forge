package main

import (
	"encoding/json"
	"log"

	"biscuitExample/authz"
	"biscuitExample/dblogic"
)

func main() {

	dbInstance, err := dblogic.InitDb()
	if err != nil {
		log.Fatalf("Error when initializing db: %s", err.Error())
	}
	defer dbInstance.Close()

	// TODO: Probably be more fun to make this a command line option..
	userId := 4
	reponame := "Charlie"
	action := authz.Read
	reqDetails, err := dblogic.GatherRequestDetails(userId, reponame, dbInstance)
	if err != nil {
		log.Fatalf("Error when gathering user details from DB: %s",
			err.Error())
	}

	prettyBytes, err := json.MarshalIndent(reqDetails, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal userDetails: %s", err.Error())
	}
	log.Printf("reqDetails are: \n%s\n == END REQ_DETAILS ==", string(prettyBytes))

	tokenIssuer, err := authz.NewTokenIssuer()
	if err != nil {
		log.Fatalf("Error when creating biscuit token issuer: %s",
			err.Error())
	}

	biscuitToken, err := tokenIssuer.IssueToken(userId)
	if err != nil {
		log.Fatalf("Error when issuing biscuit token: %s", err.Error())
	}

	// Some other attenuations to try are:
	// check if repo("repo:3")
	// check if operation($action, $repo), $action == "action:read"
	attenuationStr := "check if time($date), $date <= 2100-03-30T19:00:10Z"

	biscuitToken, err = authz.AttenuateBiscuit(biscuitToken, attenuationStr)
	if err != nil {
		log.Fatalf("Error when attenuating biscuit token: %s", err.Error())
	}

	log.Printf("biscuit token details are: \n%s\n== END BISCUIT DETAILS==", biscuitToken)

	hasPermission, err := authz.CheckAuthz(biscuitToken, tokenIssuer.PublicRoot, reqDetails, action)
	if err != nil {
		log.Fatalf("Error when checking authorization: %s", err.Error())
	}
	log.Printf("hasPermission is: %t", hasPermission)
}
