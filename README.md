# Mog : MongoDB + Go, Simplified

A set Of convenience tools that work with the offical Go driver From MongoDB.  
Intermix mog code with code that directly uses the Mongo driver.  

**Inspired By MGO Driver** 

Status - Passing all tests in mog_test.go, aggregate_test.go, and examples_test.go  
  
## Install & Use
```
go get github.com/txjmp/mog
import "github.com/txjmp/mog"
mog1 := mog.NewMog(ctx, db, collectionName)
```
**[GoDoc](https://godoc.org/github.com/txjmp/mog)**

See mog_test.go, aggregate_test.go and examples_test.go for complete examples.

## Quick Start
```
client, err = mongo.Connect(ctx, clientOptions)
db := client.Database("demo")
mog1 := NewMog(ctx, db, "property")

mog1.Omit("notes", "contacts")   // exclude notes and contacts from result
// or use mog.Keep to specify fields to include
criteria := bson.M{
	"date_added": bson.M{"$gte": "2019-01-01"},
}
mog1.Find(criteria, "city", "-date_added")  // results sorted by city, date_added (descending)
var prop Property
for mog1.Next(&prop) {
	fmt.Println(prop.City, prop.DateAdded, prop.Address)
}
```
## Aggregation Methods
There are a set of methods that handle aggregation processing. Some of these methods are designed for convenience at the expensive of flexibility. If the methods don't provide exactly what is needed there are 2 options:
1. Add stages directly to the mog.AggPipeline slice using append (must be bson.M type)
2. Work directly with Mongo driver not using any of the Mog methods  

**see aggregate_test.go for examples**
```
AggStart() - makes the AggPipeline slice 
AggKeep() - adds a $project stage, specifies fields passed to next stage
AggOmit() - adds a $project stage, specifies fields not passed to next stage
AggSort() - adds a $sort stage
AggLookupId() - adds $lookup and $unwind stages, using fromCollection primary key (_id) field to join
AggTotal() - adds $group stage, computes group count and group sum for each field specified
AggStage() - adds a stage of your making to AggPipeline
AggRun() - executes the aggregation, iterate thru results using .Next(&target) loop
AggRunAll() - executes the aggregation, all results decoded into target slice
AggShowPipeline() - displays the stages (for debugging)
```
## CSV Methods
There are a set of methods for exporting and importing data via csv files. Some of these methods are designed for convenience at the expensive of flexibility. Data is not directly imported into or exported from the collection.  
**see examples_test.go for csv examples**
```
CsvOutStart() - creates the export file and csv writer
CsvWrite() - writes a record
CsvOutDone() - flushes the csv writer and closes the output file
CsvInStart() - opens the import file and creates the csv reader
CsvVerifyHeaders() - verifies expected headers match input file headers
CsvRead() - reads a record
CsvGetVal() - provides accurate method of getting the correct value from an input record
CsvInDone() - closes the input file
CsvReadAll() - opens,reads,closes entire file and returns [][]string
```
## Mog Type
```
type Mog struct {
	ctx            context.Context
	db             *mongo.Database
	collection     *mongo.Collection
	collectionName string
	projectFlds    bson.M             // flds to be kept or omitted, use .KeepFlds or .OmitFlds to load
	bulkWrites     []mongo.WriteModel // Used by BulK.. methods
	iter           *mongo.Cursor
	iterErr        error
	limit          int64
	upsert         bool // if true, Update will add docs not matching criteria
	csvFile        *os.File
	csvWriter      *csv.Writer
	csvReader      *csv.Reader
	AggPipeline		[]bson.M
}	
```
## Mog Methods
See [GoDoc](https://godoc.org/github.com/txjmp/mog) or mog.go for details.  
```
mog := NewMog(ctx, db, ...collectionName) - create new instance of Mog
mog.SetCollection(collectionName)      - change collection
mog.SetLimit(limit int64)              - limit results, resets after execution
mog.KeepFlds(fld1, fld2, ...)          - specify flds to return in Find results
mog.OmitFlds(fld1, fld2, ...)          - specify flds to omit from Find results
mog.Find(criteria, ...sortFlds)        - creates iterator (cursor), nil criteria returns all docs
mog.Next(&doc)                         - use after Find, loads target with next doc from result
mog.FindAll(criteria, docs, ...sortFlds) - works same as Find(), except all results are loaded into docs slice
mog.IterErr() error					     - returns iterator (cursor) error after completing Find/Next process
mog.FindOne(criteria, &doc, ...sortFlds) - loads doc with 1st result, sortFlds optional
mog.FindId(docId, &doc) 				 - loads doc with result having matching id
mog.Count(criteria) 					 - returns count of docs matching criteria
mog.Update(criteria, update)  			 - update all docs matching criteria using update object
mog.Replace(criteria, newDoc)  			 - replace 1st doc matching criteria with newDoc
mog.Upsert()						     - turn upsert option on for updates, resets after execution
mog.Insert(doc1, doc2, ...)  			 - insert 1 or more docs
mog.BulkStart(size int)					 - start bulk process, size is estimated count of inserts + updates
mog.BulkAddInsert(doc interface{}) 		 - append doc to be inserted to mog.BulkWrites slice
mog.BulkAddUpdate(criteria, update interface{}) - append criteria and update to mog.BulkWrites slice
mog.BulkWrite()			                 - apply inserts & updates stored in mog.BulkWrites slice
csv input/output methods                 - see section above
aggregate methods                        - see section above
```
