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
