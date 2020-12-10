package mog

import (
	"context"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type m bson.M // for brevity

type Location struct {
	Id           string `bson:"_id" json:"id"`
	LocationName string `bson:"location_name" json:"locationName"`
}

type Property struct {
	Id         string   `bson:"_id"`
	LocationId string   `bson:"location_id"`
	Address    string   `bson:"address"`
	City       string   `bson:"city"`
	St         string   `bson:"st"`
	DateAdded  string   `bson:"date_added"` // yyyy-mm-dd
	Notes      []string `bson:"notes"`
	SumFld1    int      `bson:"sum_fld1"`
	SumFld2    float64  `bson:"sum_fld2"`
}

func Test_Mog(t *testing.T) {
	var err error
	var criteria m // see m type above
	var prop Property
	var count int64

	ctx := context.Background()
	clientOptions := options.Client()
	clientOptions.ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil || client == nil {
		t.Fatal("Mongo Connect Failed", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("demo")

	db.Collection("property").Drop(ctx)

	// ===================================================================================
	//   Add properties using Bulk process.
	//   Bulk is more efficient than Insert when adding large numbers of documents.
	// ===================================================================================
	newProps := []Property{
		{Id: NewDocId(), Address: "200 Willow Rd", City: "Wonder", St: "MT", LocationId: "122", DateAdded: "2018-03-11"},
		{Id: NewDocId(), Address: "321 Angel Way", City: "Wonder", St: "MT", LocationId: "145", DateAdded: "2019-04-04"},
		{Id: NewDocId(), Address: "1950 Hangover", City: "Las Vegas", St: "NV", LocationId: "97", DateAdded: "2017-07-29"},
	}
	mog1 := NewMog(ctx, db, "property")
	mog1.BulkStart(len(newProps))
	for _, prop := range newProps {
		mog1.BulkAddInsert(prop)
	}
	count, err = mog1.BulkWrite()
	if err != nil {
		t.Fatal("BulkWrite Inserts Failed", err)
	}
	fmt.Println("bulkLoad insert count", count)

	// ===================================================================================
	//   Add properties using Insert.
	// ===================================================================================
	newProp1 := Property{Id: "111", Address: "458 Hunker", City: "Levellear"}
	newProp2 := Property{Id: "222", Address: "876 Down", City: "Okobear"}
	err = mog1.Insert(newProp1, newProp2)
	if err != nil {
		t.Fatal("Insert Failed", err)
	}
	fmt.Println("insert successful")

	// ===================================================================================
	//   Update properties based on criteria.
	// ===================================================================================
	criteria = m{"location_id": ""}
	update := m{"$set": m{"location_id": "0"}}
	count, err = mog1.Update(criteria, update)
	if err != nil || count != 2 {
		t.Fatal("Update Failed", err, count)
	}
	fmt.Println("update count", count)

	// ===================================================================================
	//   Update using doc id
	// ===================================================================================
	update = m{"$set": m{"location_id": "300"}}
	err = mog1.UpdateId("222", update)
	if err != nil {
		t.Fatal("UpdateId Failed", err)
	}
	fmt.Println("updateId successful")

	// ===================================================================================
	//   Find (nil criteria), sorted, limit results, and iterate results.
	// ===================================================================================
	mog1.Omit("city", "location_id")
	mog1.SetLimit(2) // limit result set to 2 docs
	mog1.Find(nil, "address")
	var returnCnt int
	for mog1.Next(&prop) {
		if prop.City != "" || prop.LocationId != "" {
			t.Fatal("Omit Fields Failed", err)
		}
		fmt.Println(prop)
		returnCnt++
	}
	if mog1.IterErr() != nil || returnCnt != 2 {
		t.Fatal("Find/Iterate Failed", err, returnCnt)
	}
	fmt.Println("find/iterated successful")

	// ===================================================================================
	//   FindAll, results placed into slice.
	// ===================================================================================
	mog1.Keep("address", "city", "location_id")
	var result []Property
	err = mog1.FindAll(nil, &result, "city")
	if mog1.IterErr() != nil {
		t.Fatal("FindAll Failed", err)
	}
	for i, prop := range result {
		fmt.Println(i, prop.Address, prop.City, prop.St, prop.LocationId)
		if prop.St != "" || prop.Address == "" {
			t.Fatal("Keep Fields Failed", err)
		}
	}
	fmt.Println("findAll successful")

	// ===================================================================================
	//   Test Not Found condition
	// ===================================================================================
	err = mog1.FindId("xxxxx", &prop)
	if err != mongo.ErrNoDocuments {
		t.Fatal("Not Found Test Failed", err)
	}
	fmt.Println("test not found successful")

	// ==========================================================================================
	//   FindOne, return 1st doc using criteria and sort
	// ==========================================================================================
	criteria = m{"city": "Wonder"}
	err = mog1.FindOne(criteria, &prop, "-address")
	if err != nil || prop.Address != "321 Angel Way" {
		fmt.Printf("%+v", prop)
		t.Fatal("FindOne Failed", err)
	}
	fmt.Println("FindOne successful")

	// ==========================================================================================
	//  Count docs meeting criteria.
	// ==========================================================================================
	count, err = mog1.Count(m{"st": "MT"})
	if err != nil || count != 2 {
		t.Fatal("Count Failed", err)
	}
	fmt.Println("count successful")
}
