// Mog - Mongo Made More Good

package mog

// mog := NewMog(db, ...collectionName)  	// db is *mongo.Database, collectionName is optional
// mog.SetCollection(collectionName)		// change collection
// mogSetLimit(limit int64)					// set limit value, resets after execution
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
// mog.BulkWrite()			// apply inserts/updates stored in mog.BulkWrites, returns total of inserts + updates

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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
}

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

func (mog *Mog) SetCollection(collectionName string) {
	mog.collection = mog.db.Collection(collectionName)
	mog.collectionName = collectionName
}

func (mog *Mog) SetLimit(limit int64) {
	mog.limit = limit
}

func (mog *Mog) Upsert() {
	mog.upsert = true
}

// Find creates a cursor (iterator) for docs meeting criteria
// mog.iter = cursor
// mog.Next uses mog.iter to iterate thru results
// Use mog.KeepFlds or mog.OmitFlds to load mog.projectFlds.
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

// FindAll loads all matching docs into docs
// docs parm should be address of target slice where results will be loaded
// Otherwise, works same as Find()
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

// *************************************************************************************
// ErrNoDocuments means that the filter did not match any documents in the collection
// if err == mongo.ErrNoDocuments {
//	 ...
// }
// *************************************************************************************

// FindOne returns the 1st doc found based on criteria and sort order
// doc parm should be address of target where result will be loaded
func (mog *Mog) FindOne(criteria interface{}, doc interface{}, sortFlds ...string) error {
	findOptions := options.FindOne()
	if len(sortFlds) > 0 {
		sortOrder := CreateSortOrder(sortFlds)
		findOptions.SetSort(sortOrder)
	}
	err := mog.collection.FindOne(mog.ctx, criteria, findOptions).Decode(doc)
	return err
}

// FindId returns doc with matching _id
// doc parm should be address of target where result will be loaded
func (mog *Mog) FindId(docId interface{}, doc interface{}) error {
	criteria := bson.M{"_id": docId}
	err := mog.collection.FindOne(mog.ctx, criteria).Decode(doc)
	return err
}

// Next loads next doc returned by iterator (cursor) created by previous Find
// doc should be address of target where next result will be loaded
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

func (mog *Mog) IterErr() error {
	return mog.iterErr
}

func (mog *Mog) CloseIter() error {
	err := mog.iter.Close(mog.ctx)
	return err
}

// Count returns count of docs matching criteria
func (mog *Mog) Count(criteria interface{}) (int64, error) {
	countOptions := options.Count()
	if mog.limit > 0 { // limit the number of docs to count
		countOptions.SetLimit(mog.limit)
		mog.limit = 0
	}
	count, err := mog.collection.CountDocuments(mog.ctx, criteria, countOptions)
	return count, err
}

// Update updates docs matching criteria using update
func (mog *Mog) Update(criteria, update interface{}) (int64, error) {
	updateOptions := options.Update()
	if mog.upsert { // if true, insert docs not matching criteria
		updateOptions.SetUpsert(true)
		mog.upsert = false
	}
	changeInfo, err := mog.collection.UpdateMany(mog.ctx, criteria, update, updateOptions)
	return changeInfo.ModifiedCount + changeInfo.UpsertedCount, err
}

// Replace replaces 1st doc matching criteria, with newDoc
func (mog *Mog) Replace(criteria, newDoc interface{}) error {
	replaceOptions := options.Replace()
	if mog.upsert { // insert new doc, if no doc found matching criteria
		replaceOptions.SetUpsert(true)
		mog.upsert = false
	}
	_, err := mog.collection.ReplaceOne(mog.ctx, criteria, newDoc, replaceOptions)
	return err
}

// UpdateId updates doc with matching id
func (mog *Mog) UpdateId(docId, update interface{}) error {
	criteria := bson.M{"_id": docId}
	_, err := mog.collection.UpdateOne(mog.ctx, criteria, update)
	return err
}

// Insert adds 1 or more documents to collection (use Bulk for large number of inserts)
func (mog *Mog) Insert(docs ...interface{}) error {
	_, err := mog.collection.InsertMany(mog.ctx, docs)
	return err
}

// BulkStart called at beginning of bulk write process, size is estimated # of updates
func (mog *Mog) BulkStart(size int) {
	mog.bulkWrites = make([]mongo.WriteModel, 0, size)
}

// BulkAddInsert adds documents to be inserted to mog.BulkWrites
func (mog *Mog) BulkAddInsert(doc interface{}) {
	model := mongo.NewInsertOneModel()
	model.SetDocument(doc)
	mog.bulkWrites = append(mog.bulkWrites, model)
}

// BulkAddUpdate adds matching criteria and update doc to mog.BulkWrites
func (mog *Mog) BulkAddUpdate(criteria, update interface{}) {
	model := mongo.NewUpdateManyModel()
	model.SetFilter(criteria)
	model.SetUpdate(update)
	mog.bulkWrites = append(mog.bulkWrites, model)
}

// BulkWrite executes bulk write using entries in mog.BulkWrites
func (mog *Mog) BulkWrite() (int64, error) {
	result, err := mog.collection.BulkWrite(mog.ctx, mog.bulkWrites)
	mog.bulkWrites = nil
	return result.InsertedCount + result.ModifiedCount, err
}

// Keep loads ProjectFlds with map of flds to be kept in Find results
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

// Omit loads ProjectFlds with map of flds to be omitted from Find results
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

// CreateSorteOrder returns slice of elements defining sort order
// keyFlds are field names to be sorted in order
// keys to be sorted in descending order begin with a minus sign "-"
// the val for each key is 1 for ascending, -1 for descending
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

func NewDocId() string {
	return primitive.NewObjectID().Hex()
}
