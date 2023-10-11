package mongohs

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/hiveot/hub/pkg/bucketstore"
)

type MongoBucket struct {
	clientID string
	bucketID string
	//
	collection *mongo.Collection
}

// Close the bucket and release its resources
func (bucket *MongoBucket) Close() (err error) {

	return fmt.Errorf("not implemented")
}

func (bucket *MongoBucket) Cursor() (cursor bucketstore.IBucketCursor) {
	//var mongoCursor = nil
	//ctx := context.Background()
	//mongoCursor, err := bucket.collection.Find(ctx, nil)//filter)
	//if err != nil {
	//	logrus.Warning(err)
	//	return nil
	//}
	//mongoCursor

	//bucketCursor := NewMongoCursor(bucket.bucketID, mongoCursor)
	return nil
}

func (bucket *MongoBucket) Delete(key string) error {
	ctx := context.Background()
	filter := bson.D{{"key", key}}
	_, err := bucket.collection.DeleteOne(ctx, filter)
	return err
}

func (bucket *MongoBucket) Get(key string) (val []byte, err error) {
	ctx := context.Background()
	filter := bson.D{{"key", key}}
	res := bucket.collection.FindOne(ctx, filter)
	err = res.Decode(&val)
	return val, err
}
func (bucket *MongoBucket) GetMultiple(keys []string) (docs map[string][]byte, err error) {
	return nil, fmt.Errorf("not implemented")
}

func (bucket *MongoBucket) ID() string {
	return bucket.bucketID
}

func (bucket *MongoBucket) Info() (info *bucketstore.BucketStoreInfo) {
	ctx := context.Background()
	// Info returns mongoClient statistics
	nrEntries, _ := bucket.collection.CountDocuments(ctx, bson.D{})

	info = &bucketstore.BucketStoreInfo{
		DataSize:  -1,
		Engine:    bucketstore.BackendMongoDB,
		Id:        bucket.bucketID,
		NrRecords: nrEntries,
	}
	return info
}

func (bucket *MongoBucket) Set(key string, doc []byte) error {
	//ctx := context.Background()
	//filter := bson.D{{"key", key}}
	//timestamp := primitive.NewDateTimeFromTime(createdTime)
	//evBson := bson.M{
	//	TimeStampField: timestamp,
	//	"metadata":     bson.M{"thingID": actionValue.ThingID, "name": actionValue.ID},
	//	"name":         actionValue.ID,
	//	"thingID":      actionValue.ThingID,
	//	"value":        actionValue.ValueJSON,
	//	"created":      actionValue.Created,
	//}
	//res, err := bucket.collection.UpdateOne(ctx, filter, evBson)
	return fmt.Errorf("not implemented")

}

func (bucket *MongoBucket) SetMultiple(docs map[string][]byte) (err error) {
	return fmt.Errorf("not implemented")
}

func NewMongoBucket(clientID, bucketID string, collection *mongo.Collection) *MongoBucket {
	bucket := &MongoBucket{
		clientID:   clientID,
		bucketID:   bucketID,
		collection: collection,
	}
	return bucket
}
