# Mog : MongoDB + Go, Simplified

A Set Of Convenience Tools That Use Offical Go Driver From MongoDB  

**Inspired By MGO Driver** 

See mog_test.go for complete examples.

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
   







```
// Used In Examples Below
type Property struct {
	Id             string    `bson:"_id" json:"id"`
	Address        string    `bson:"address" json:"address"`
	City           string    `bson:"city" json:"city"`
	St             string    `bson:"st" json:"st"`
    Notes          []string  `bson:"notes" json:"notes"`
}
```

```
FindAll(criteria interface{}, docs interface{}, sortflds ...string) error
criteria: match criteria, typically bson.M, nil = all docs
docs: 
```
  
```
// return all docs in "property" collection, store decoded results sorted by "address"
mog := NewMog(ctx, db, "property") 
mog.Omit("notes")  // don't return value for "notes" field
var result []Property
err := mog.FindAll(nil, &result, "address")
```

### Simplifies
* Sorting
* Return Values Only for Specific Fields
* Omit Values for Specific Fields
* 