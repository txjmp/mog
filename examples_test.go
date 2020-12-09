package mog

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/*  following defined in mog_test.go

type Location struct {
	Id           string `bson:"_id" json:"id"`
	LocationName string `bson:"location_name" json:"locationName"`
}

type Property struct {
	Id         string   `bson:"_id" json:"id"` // json tags shown for example, not used here
	LocationId string   `bson:"location_id" json:"locationId"`
	Address    string   `bson:"address" json:"address"`
	City       string   `bson:"city" json:"city"`
	St         string   `bson:"st" json:"st"`
	DateAdded  string   `bson:"date_added" json:"dateAdded"` // yyyy-mm-dd
	Notes      []string `bson:"notes" json:"notes"`
}
*/
func ExampleMog() {
	var err error

	ctx := context.Background()
	clientOptions := options.Client()
	clientOptions.ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil || client == nil {
		log.Fatal("Mongo Connect Failed", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("demo")

	db.Collection("property").Drop(ctx)
	db.Collection("location").Drop(ctx)

	// ===================================================================================
	//   Add locations for test data using Insert method.
	// ===================================================================================
	locs := []Location{
		{Id: "7", LocationName: "Northwest"},
		{Id: "10", LocationName: "Southwest"},
	}
	mog1 := NewMog(ctx, db, "location")
	mog1.Insert(locs[0], locs[1])

	// ===================================================================================
	//   Add properties for test data using Bulk methods.
	// ===================================================================================
	mog1.SetCollection("property")
	props := []Property{
		{Id: NewDocId(), Address: "200 Willow Rd", City: "Wonder", St: "MT", LocationId: "7", DateAdded: "2018-03-11"},
		{Id: NewDocId(), Address: "321 Angel Way", City: "Wonder", St: "MT", LocationId: "7", DateAdded: "2019-04-04"},
		{Id: NewDocId(), Address: "1950 Hangover", City: "Las Vegas", St: "NV", LocationId: "10", DateAdded: "2017-07-29"},
	}
	mog1.BulkStart(len(props))
	for _, prop := range props {
		mog1.BulkAddInsert(prop)
	}
	mog1.BulkWrite()

	// ===================================================================================
	//  Create map containing location data. Key is location_id.
	// ===================================================================================
	mog1.SetCollection("location")
	locationMap := make(map[string]*Location)
	mog1.Find(nil)
	loc := new(Location)
	for mog1.Next(loc) {
		locationMap[loc.Id] = loc
		loc = new(Location)
	}

	// ===================================================================================
	//  Create CSV file.
	// ===================================================================================
	mog1.SetCollection("property")

	filePath := "/home/jay/test/mog_props.csv"

	err = mog1.CsvOutStart(filePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	headers := []string{"Location", "Address", "City"}
	mog1.CsvWrite(headers)

	mog1.Find(nil)
	var prop Property
	for mog1.Next(&prop) {
		locName := locationMap[prop.LocationId].LocationName
		record := []string{locName, prop.Address, prop.City}
		mog1.CsvWrite(record)
	}
	err = mog1.CsvOutDone()

	// ===================================================================================
	//  Read CSV file.
	// ===================================================================================
	err = mog1.CsvInStart(filePath)
	if err != nil {
		fmt.Println("csvinstart failed", err)
		return
	}
	var rec []string
	fmt.Println("--- result 1 ------------------")
	for {
		rec, err = mog1.CsvRead()
		if err != nil {
			break
		}
		fmt.Println(strings.Join(rec, "|"))
	}
	if err != io.EOF {
		fmt.Println("csvread failed", err)
	}
	mog1.CsvInDone()
	// ===================================================================================
	//  ReadAll CSV file.
	// ===================================================================================
	records, err := mog1.CsvReadAll(filePath)
	if err != nil {
		panic(err)
	}
	fmt.Println("--- result 2 ------------------")
	for _, rec := range records {
		fmt.Println(strings.Join(rec, "|"))
	}

	// Output:
	// --- result 1 ------------------
	// Location|Address|City
	// Northwest|200 Willow Rd|Wonder
	// Northwest|321 Angel Way|Wonder
	// Southwest|1950 Hangover|Las Vegas
	// --- result 2 ------------------
	// Location|Address|City
	// Northwest|200 Willow Rd|Wonder
	// Northwest|321 Angel Way|Wonder
	// Southwest|1950 Hangover|Las Vegas
}
