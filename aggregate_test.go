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
	mog1.AggShowPipeline()                                                   // for debugging
	opts := options.Aggregate().SetMaxTime(2 * time.Second)                  // see mongo driver documentation for all options
	mog1.AggRun(opts)

	//var result bson.D   // generic way to decode result record
	var result struct {
		State string `bson:"_id"`
		Count int    `bson:"count"`
	}
	for mog1.Next(&result) {
		fmt.Printf("%+v\n", result)
	}

	/*
		pipeLine := []m{
			m{"$project": m{"address": 1, "st": 1, "city": 1, "notecount": m{"$size": "$notes"}}},  // output address, city, st, notecount
			m{"$match": m{"notecount": m{"$gt": 2}}},                                               // keep docs with more than 2 notes
			m{"$sort": bson.D{{"st", 1}, {"city", 1}}},                                             // sort results by state, city - see note above
		}
		iter := collection.Pipe(pipeLine).Iter()
		defer iter.Close()

		var result struct {
			Id        string `bson:"_id"`
			State     string `bson:"st"`
			City      string `bson:"city"`
			Address   string `bson:"address"`
			NoteCount int    `bson:"notecount"`
		}
		for iter.Next(&result) {
			log.Printf("%+v", result)
		}
		if iter.Err() != nil {
			log.Println(iter.Err())
		}
	*/
	//pipeLine := mongo.Pipeline{
	//	{{"$group", bson.D{{"_id", "$st"}, {"totalPop", bson.D{{"$sum", "$pop"}}}}}},
	//	{{"$match", bson.D{{"totalPop", bson.D{{"$gte", 10*1000*1000}}}}}},
	//}

	// Output:
	// Location|Address|City
	// Northwest|200 Willow Rd|Wonder
	// Northwest|321 Angel Way|Wonder
	// Southwest|1950 Hangover|Las Vegas
}
