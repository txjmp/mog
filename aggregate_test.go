package mog

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
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
func ExampleAggregate() {
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
	//   Aggregate - count properties by state, results sorted by state
	// ===================================================================================
	mog1.AggStart()                                                          // create new pipeline slice to hold stages
	mog1.AggStage("group", bson.M{"_id": "$st", "count": bson.M{"$sum": 1}}) // add stage - count props by state
	mog1.AggSort("_id")                                                      // add stage - sort by the group id (state)
	//mog1.AggShowPipeline()                                                   // for debugging
	opts := options.Aggregate().SetMaxTime(2 * time.Second) // see mongo driver documentation for all options
	err = mog1.AggRun(opts)

	//var result bson.D   // generic way to decode result record
	var result struct {
		State string `bson:"_id"`
		Count int    `bson:"count"`
	}
	fmt.Println("--- result1 ----------------------")
	for mog1.Next(&result) {
		fmt.Println(result.State, result.Count)
	}
	// ===================================================================================
	//   Aggregate - lookup location
	// ===================================================================================
	mog1.AggStart()                                    // create new pipeline slice to hold stages
	mog1.AggLookupId("location", "location_id", "loc") // add lookup & unwind stages
	mog1.AggKeep("address", "loc")                     // add project stage
	//mog1.AggShowPipeline()                                                   // for debugging
	err = mog1.AggRun()
	if err != nil {
		panic(err)
	}
	var result2 struct {
		Id      string `bson:"_id"`
		Address string `bson:"address"`
		Loc     struct {
			LocName string `bson:"location_name"`
		} `bson:"loc"`
	}
	fmt.Println("--- result2 ----------------------")
	for mog1.Next(&result2) {
		fmt.Println(result2.Address, result2.Loc.LocName)
	}
	if mog1.IterErr() != nil {
		panic(err)
	}

	// Output:
	// --- result1 ----------------------
	// MT 2
	// NV 1
	// --- result2 ----------------------
	// 200 Willow Rd Northwest
	// 321 Angel Way Northwest
	// 1950 Hangover Southwest

}
