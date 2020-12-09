// Package Mog makes using MongoDB fun and easy. It uses the official Go driver from MongoDB.
package mog

// mog := NewMog(db, ...collectionName)  	// db is *mongo.Database, collectionName is optional
// mog.SetCollection(collectionName)		// change collection
// mog.SetLimit(limit int64)					// set limit value, resets after execution
// mog.KeepFlds(fld1, fld2, ...)  			// specify flds to return in Find results
// mog.OmitFlds(fld1, fld2, ...)  			// specify flds to omit from Find results
// mog.Find(criteria, ...sortFlds)  		// creates iterator (cursor), sortFlds optional, nil criteria returns all docs
// mog.Next(&doc)  							// use after Find, loads target with next doc from results, iter closed automatically at end, returns true if more
// mog.FindAll(criteria, docs, ...sortFlds) // works same as Find(), except all results are loaded into docs slice
// mog.IterErr() error						// returns iterator (cursor) error after completing Find/Next process
// mog.FindOne(criteria, &doc, ...sortFlds) // loads doc with 1st result, sortFlds optionals
// mog.FindId(docId, &doc) 					// loads doc with result having matching id
// mog.Count(criteria) 						// returns count of docs matching criteria
// mog.Update(criteria, update)  			// update all docs matching criteria using update object
// mog.Replace(criteria, newDoc)  			// replace 1st doc matching criteria with newDoc
// mog.Upsert()								// turn upsert option on for updates, resets after execution
// mog.Insert(doc1, doc2, ...)  			// insert 1 or more docs
// mog.BulkStart(size int)					// start bulk process, size is estimated count of inserts + updates
// mog.BulkAddInsert(doc interface{}) 		// append doc to be inserted to mog.BulkWrites slice
// mog.BulkAddUpdate(criteria, update interface{}) // append criteria and update code to mog.BulkWrites slice
// mog.BulkWrite()							// apply inserts/updates stored in mog.BulkWrites, returns total of inserts + updates
// mog.CsvOutStart(filePath)				// begin csv output
// mog.CsvWrite(record)						// write record to csv output
// mog.CsvOutDone()							// complete csv output
// mog.CsvInStart(filePath)					// begin csv input
// mog.CsvRead()							// read record from csv input
// mog.CsvInDone()							// close csv input file

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Type Mog contains almost everything.
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
	AggPipeline    []bson.M
}

// NewMog creates instance of Mog.
func NewMog(ctx context.Context, db *mongo.Database, collectionName ...string) *Mog {
	mog := Mog{
		ctx: ctx,
		db:  db,
	}
	if len(collectionName) > 0 {
		mog.collection = db.Collection(collectionName[0])
		mog.collectionName = collectionName[0]
	}
	return &mog
}

// SetCollection changes the collection used.
func (mog *Mog) SetCollection(collectionName string) {
	mog.collection = mog.db.Collection(collectionName)
	mog.collectionName = collectionName
}

// SetLimit limits the number of docs returned. Resets after execution.
func (mog *Mog) SetLimit(limit int64) {
	mog.limit = limit
}

// Upsert turns upsert option on (see MongoDB doc). Resets after execution.
func (mog *Mog) Upsert() {
	mog.upsert = true
}

// Find sets mog.iter = mongo cursor (iterator) for docs meeting criteria.
// Next() method uses mog.iter to iterate thru results.
// Use criteria parm to filter results (nil for all docs in collection).
// Use optional sortFlds to sort. Begin fieldname with "-" for descending.
func (mog *Mog) Find(criteria interface{}, sortFlds ...string) {
	findOptions := options.Find()
	if len(sortFlds) > 0 {
		sortOrder := CreateSortOrder(sortFlds)
		findOptions.SetSort(sortOrder)
	}
	if mog.projectFlds != nil {
		findOptions.SetProjection(mog.projectFlds)
	}
	if mog.limit > 0 {
		findOptions.SetLimit(mog.limit)
		mog.limit = 0
	}
	if criteria == nil {
		criteria = bson.D{{}}
	}
	cursor, _ := mog.collection.Find(mog.ctx, criteria, findOptions)
	mog.iter = cursor
}

// FindAll loads all matching docs into slice.
// Parm "docs" should be address of target slice where results will be loaded.
// Otherwise, works same as Find().
func (mog *Mog) FindAll(criteria interface{}, docs interface{}, sortFlds ...string) error {
	findOptions := options.Find()
	if len(sortFlds) > 0 {
		sortOrder := CreateSortOrder(sortFlds)
		findOptions.SetSort(sortOrder)
	}
	if mog.projectFlds != nil {
		findOptions.SetProjection(mog.projectFlds)
	}
	if mog.limit > 0 {
		findOptions.SetLimit(mog.limit)
		mog.limit = 0
	}
	if criteria == nil {
		criteria = make(bson.D, 0)
	}
	cursor, err := mog.collection.Find(mog.ctx, criteria, findOptions)
	if err != nil {
		return err
	}
	err = cursor.All(mog.ctx, docs)
	return err
}

// FindOne returns the 1st doc found based on criteria and sort order.
// Parm "doc" should be address of target where result will be loaded.
// If error == mongo.ErrNoDocuments, no docs found matching criteria.
func (mog *Mog) FindOne(criteria interface{}, doc interface{}, sortFlds ...string) error {
	findOptions := options.FindOne()
	if len(sortFlds) > 0 {
		sortOrder := CreateSortOrder(sortFlds)
		findOptions.SetSort(sortOrder)
	}
	if mog.projectFlds != nil {
		findOptions.SetProjection(mog.projectFlds)
	}
	err := mog.collection.FindOne(mog.ctx, criteria, findOptions).Decode(doc)
	return err
}

// FindId returns doc with matching _id.
// Parm "doc" should be address of target where result will be loaded.
func (mog *Mog) FindId(docId interface{}, doc interface{}) error {
	criteria := bson.M{"_id": docId}
	err := mog.collection.FindOne(mog.ctx, criteria).Decode(doc)
	return err
}

// Next loads next doc returned by mog.iter (cursor) created by previously run Find().
// Parm "doc" should be address of target where next result will be loaded.
// Returns true if more results to process, otherwise false.
// After completion, usg mog.IterErr() to get error value.
// Iterator is automatically closed after last result processed.
func (mog *Mog) Next(doc interface{}) bool {
	more := mog.iter.Next(mog.ctx)
	if !more {
		mog.iterErr = mog.iter.Err()
		mog.iter.Close(mog.ctx)
		return false
	}
	err := mog.iter.Decode(doc)
	if err != nil {
		log.Println("mog.Next decode error", mog.collectionName, err)
		mog.iterErr = err
		return false
	}
	return more
}

// IterErr returns value of mog.itererr which is set by Next() method.
func (mog *Mog) IterErr() error {
	return mog.iterErr
}

// CloseIter closes mog.iter. Use if all results not processed by Next().
func (mog *Mog) CloseIter() error {
	err := mog.iter.Close(mog.ctx)
	return err
}

// Count returns count of docs matching criteria.
func (mog *Mog) Count(criteria interface{}) (int64, error) {
	countOptions := options.Count()
	if mog.limit > 0 { // limit the number of docs to count
		countOptions.SetLimit(mog.limit)
		mog.limit = 0
	}
	count, err := mog.collection.CountDocuments(mog.ctx, criteria, countOptions)
	return count, err
}

// Update updates docs matching parm "criteria" using parm "update".
// To update all docs, criteria should be type bson.D with no elements - bson.D{}.
func (mog *Mog) Update(criteria, update interface{}) (int64, error) {
	if criteria == nil {
		return 0, errors.New("nil criteria not allowed for update")
	}
	updateOptions := options.Update()
	if mog.upsert { // if true, insert docs not matching criteria
		updateOptions.SetUpsert(true)
		mog.upsert = false
	}
	changeInfo, err := mog.collection.UpdateMany(mog.ctx, criteria, update, updateOptions)
	return changeInfo.ModifiedCount + changeInfo.UpsertedCount, err
}

// Replace replaces 1st doc matching criteria, with newDoc.
func (mog *Mog) Replace(criteria, newDoc interface{}) error {
	replaceOptions := options.Replace()
	if mog.upsert { // insert new doc, if no doc found matching criteria
		replaceOptions.SetUpsert(true)
		mog.upsert = false
	}
	_, err := mog.collection.ReplaceOne(mog.ctx, criteria, newDoc, replaceOptions)
	return err
}

// UpdateId updates doc with matching id.
func (mog *Mog) UpdateId(docId, update interface{}) error {
	criteria := bson.M{"_id": docId}
	_, err := mog.collection.UpdateOne(mog.ctx, criteria, update)
	return err
}

// Insert adds 1 or more documents to collection (use Bulk for large number of inserts).
func (mog *Mog) Insert(docs ...interface{}) error {
	_, err := mog.collection.InsertMany(mog.ctx, docs)
	return err
}

// BulkStart called at beginning of bulk write process, size is estimated # of updates.
func (mog *Mog) BulkStart(size int) {
	mog.bulkWrites = make([]mongo.WriteModel, 0, size)
}

// BulkAddInsert adds documents to be inserted to mog.BulkWrites.
func (mog *Mog) BulkAddInsert(doc interface{}) {
	model := mongo.NewInsertOneModel()
	model.SetDocument(doc)
	mog.bulkWrites = append(mog.bulkWrites, model)
}

// BulkAddUpdate adds matching criteria and update doc to mog.BulkWrites.
func (mog *Mog) BulkAddUpdate(criteria, update interface{}) {
	model := mongo.NewUpdateManyModel()
	model.SetFilter(criteria)
	model.SetUpdate(update)
	mog.bulkWrites = append(mog.bulkWrites, model)
}

// BulkWrite executes bulk write using entries in mog.BulkWrites.
func (mog *Mog) BulkWrite() (int64, error) {
	result, err := mog.collection.BulkWrite(mog.ctx, mog.bulkWrites)
	mog.bulkWrites = nil
	return result.InsertedCount + result.ModifiedCount, err
}

// Keep loads ProjectFlds with map of flds to be kept in Find results.
// Call Keep with no parms to reset to all fields.
// Use Keep or Omit, not both.
func (mog *Mog) Keep(flds ...string) {
	if len(flds) == 0 { // allows reuse of same mog object when all fields should be returned
		mog.projectFlds = nil
		return
	}
	mog.projectFlds = make(bson.M)
	for _, fld := range flds {
		mog.projectFlds[fld] = 1
	}
}

// Omit loads ProjectFlds with map of flds to be omitted from Find results.
// Call Omit with no parms to reset to all fields.
// Use Omit or Keep, not both.
func (mog *Mog) Omit(flds ...string) {
	if len(flds) == 0 { // allows reuse of same mog object when no fields should be omitted
		mog.projectFlds = nil
		return
	}
	mog.projectFlds = make(bson.M)
	for _, fld := range flds {
		mog.projectFlds[fld] = 0
	}
}

// --- CSV Methods ----------------------------------------------------

// CsvOutStart creates csv output file and csv writer. Comma is field delimiter.
// Optional useCRLF indicates records should end with \r\n. Default terminator is \n.
func (mog *Mog) CsvOutStart(filePath string, useCRLF ...bool) error {
	var err error
	mog.csvFile, err = os.Create(filePath)
	if err != nil {
		return err
	}
	mog.csvWriter = csv.NewWriter(mog.csvFile)
	if len(useCRLF) > 0 {
		mog.csvWriter.UseCRLF = useCRLF[0]
	}
	return nil
}

// CsvInStart opens input file and creates csv reader.
func (mog *Mog) CsvInStart(filePath string) error {
	var err error
	mog.csvFile, err = os.Open(filePath)
	if err != nil {
		return err
	}
	mog.csvReader = csv.NewReader(mog.csvFile)
	return nil
}

// CsvWrite writes record using csv writer created by CsvOutStart.
func (mog *Mog) CsvWrite(record []string) {
	mog.csvWriter.Write(record)
}

// CsvRead reads record using csv reader created by CsvInStart.
// After all data is read, returns nil, io.EOF.
func (mog *Mog) CsvRead() ([]string, error) {
	record, err := mog.csvReader.Read()
	return record, err
}

// CsvOutDone flushes csv writer and closes output file.
// Any error that occurred during write or flush steps is returned.
func (mog *Mog) CsvOutDone() error {
	mog.csvWriter.Flush()
	mog.csvFile.Close()
	return mog.csvWriter.Error()
}

// CsvInDone closes input csv file.
func (mog *Mog) CsvInDone() {
	mog.csvFile.Close()
}

// --- Aggregate Methods ----------------------------------------------------

// AggStart makes new AggPipeline slice.
func (mog *Mog) AggStart() {
	mog.AggPipeline = make([]bson.M, 0, 10)
}

// AggStage adds a stage to AggPipeline.
// Parm "op" is operation ("match", "group", etc.)
// Parm "opParms" is map of values used for the operations.
// Ex: AggStage("group", bson.M{"_id": "$st", "count": bson.M{"$sum": 1}})
func (mog *Mog) AggStage(op string, opParms bson.M) {
	opCode := "$" + op
	stage := bson.M{opCode: opParms}
	mog.AggPipeline = append(mog.AggPipeline, stage)
}

// AggLookupId adds $lookup and $unwind stages to AggPipeline.
// ForeignField is assumed to be "_id".
// There should be 1 doc in fromCollection where _id value matches localField value.
func (mog *Mog) AggLookupId(fromCollection, localField, asName string) {
	mog.AggStage("lookup", bson.M{
		"from":         fromCollection,
		"localField":   localField,
		"foreignField": "_id",
		"as":           asName,
	})
	mog.AggPipeline = append(mog.AggPipeline, bson.M{"$unwind": "$" + asName})
}

// AggKeep works basically the same as Keep method (used for Find operations).
// It determines what fields are passed along in the pipeline.
// A $project stage is added to AggPipeline.
func (mog *Mog) AggKeep(flds ...string) {
	projectFlds := make(bson.M)
	for _, fld := range flds {
		projectFlds[fld] = 1
	}
	mog.AggStage("project", projectFlds)
}

// AggSort adds sort stage to AggPipeline.
// More convenient than using AggStage method.
func (mog *Mog) AggSort(keyFlds ...string) {
	var sortOrder bson.D
	sortOrder = CreateSortOrder(keyFlds)
	stage := bson.M{"$sort": sortOrder}
	mog.AggPipeline = append(mog.AggPipeline, stage)
}

// AggRun executes the collection.Aggregate method using the AggPipeline.
// Options can be set using optional mongo/options.AggregateOptions (see Mongo driver documentation).
// The iterator, mog.iter, is loaded with the results cursor.
// Use mog.Next() to iterate thru the results.
// After complete, use mog.IterErr() to check for errors.
func (mog *Mog) AggRun(aggOptions ...*options.AggregateOptions) error {
	opts := new(options.AggregateOptions)
	if len(aggOptions) > 0 {
		opts = aggOptions[0]
	}
	cursor, err := mog.collection.Aggregate(mog.ctx, mog.AggPipeline, opts)
	mog.iter = cursor
	return err
}

// AggShowPipeline displays the aggregation pipeline stages(mog.AggPipeline).
// Useful for debugging.
func (mog *Mog) AggShowPipeline() {
	fmt.Println("--- Aggregate Pipeline Stages ----------------------------")
	for _, stage := range mog.AggPipeline {
		fmt.Printf("%+v\n", stage)
	}
	fmt.Println()
}

// CreateSorteOrder returns slice of bson elements (type bson.D) defining sort order.
// Parm "keyFlds" are field names to be sorted in order of precedence.
// Keys to be sorted in descending order begin with a minus sign "-".
func CreateSortOrder(keyFlds []string) bson.D {
	sortOrder := make(bson.D, len(keyFlds))
	for i, keyFld := range keyFlds {
		if keyFld[0:1] == "-" {
			sortOrder[i] = bson.E{Key: keyFld[1:], Value: -1} // descending, remove leading minus sign
		} else {
			sortOrder[i] = bson.E{Key: keyFld, Value: 1} // ascending
		}
	}
	return sortOrder
}

/*
	A note about bson.D & bson.E
	bson.D is a slice of elements
	bson.E is an element:
		type E struct {
			Key   string
			Value interface{}
		}
*/

// NewDocId returns a unique 24 char hexadecimal value used for new doc ids.
func NewDocId() string {
	return primitive.NewObjectID().Hex()
}
