// Package mongohs with MongoDB based history mongoClient
// This implements the HistoryStore.proto API
package mongohs

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/hiveot/hub/pkg/bucketstore"
)

const TimeStampField = "timestamp"

// UNFINISHED - DOESNT WORK
//
// Unfortunately the MongoDB collection cursor doesn't meet the needs of a bucket cursor.
// Not sure if there is a good reason to figure out a workaround.
//
// stuff to read:
//  https://www.mongodb.com/databases/key-value-database and
//  https://www.mongodb.com/community/forums/t/using-mongodb-as-a-key-value-store-which-fetch-multiple-keys-together/108686/6

// MongoBucketStore uses MongoDB to mongoClient client buckets in a mongoDB database.
// This mongoClient uses a database per client and a collection for each bucket.
// Optionally a time-series collection can be used instead of a regular collection.
//
// This implements the bucketstore.IBucketStore interface with the IBucket interface
// for each collection.
type MongoBucketStore struct {
	clientID string // used as the database name
	dbPass   string // mongodb password
	dbURL    string // mongodb connection url

	// Use time series for the bucket collection
	useTimeSeries bool

	// Client connection to the data mongoClient
	mongoClient *mongo.Client
	// database instance
	storeDB *mongo.Database

	//bucketCollection map[string]*mongo.Collection

	// startTime is the time this service started
	//startTime time.Time
}

// GetBucket returns a bucket to use
// This creates a new collection if it doesn't yet exist.
func (srv *MongoBucketStore) GetBucket(bucketID string) bucketstore.IBucket {
	var err error
	ctx := context.Background()
	var filter bson.M
	var names []string
	names, err = srv.storeDB.ListCollectionNames(ctx, filter)
	if len(names) == 0 && err == nil {
		// create the collection
		err = srv.createBucketCollection(bucketID)
	}

	collection := srv.storeDB.Collection(bucketID)

	mongoBucket := NewMongoBucket(srv.clientID, bucketID, collection)
	return mongoBucket
}

// Create a new collection for the given bucket
func (srv *MongoBucketStore) createBucketCollection(bucketID string) (err error) {
	ctx := context.Background()
	co := &options.CreateCollectionOptions{}

	logrus.Warningf("Creating the collection for '%s'", bucketID)

	if srv.useTimeSeries {
		// configure the time series options
		// A granularity of hours is best, if one sample per minute is received per sensor.
		// choosing seconds will increase read times as many internal buckets need to be read.
		// choosing many hours will increase write times if more samples are received as many steps are needed to add to
		// a bucket.
		// See also this slideshare on choosing granularity:
		//   https://www.slideshare.net/mongodb/mongodb-for-time-series-data-setting-the-stage-for-sensor-management
		//
		// TBD: optimize granularity per bucket in case buckets have different data types and frequency.
		// setting this to hours will reduce query memory consumption

		// prepare options
		tso := &options.TimeSeriesOptions{
			TimeField: TimeStampField,
		}
		tso.SetMetaField("metadata")
		tso.SetGranularity("hours")
		co.SetTimeSeriesOptions(tso)

		err = srv.storeDB.CreateCollection(ctx, bucketID, co)

		// secondary index to improve sort speed using metadata.name, time
		// https://www.mongodb.com/docs/v6.0/core/timeseries/timeseries-secondary-index/
		c := srv.storeDB.Collection(bucketID)
		nameIndex := mongo.IndexModel{Keys: bson.D{
			{"metadata.name", 1},
			{"timestamp", -1},
		}, Options: nil}
		indexName, err2 := c.Indexes().CreateOne(ctx, nameIndex)
		_ = indexName
		err = err2

	} else {
		//use the default options
		err = srv.storeDB.CreateCollection(ctx, bucketID, co)
	}
	return err
}

// Open connects to the DB server.
// This will setup the database if the collections haven't been created yet.
// Connect must be called before any other method, including Setup or Delete
func (srv *MongoBucketStore) Open() (err error) {
	ctx := context.Background()
	logrus.Infof("Connecting to the mongodb database on '%s'", srv.dbURL)
	if srv.mongoClient != nil {
		err = fmt.Errorf("store for client '%s' already started", srv.clientID)
		logrus.Error(err)
		return err
	}
	srv.mongoClient, err = mongo.NewClient(options.Client().ApplyURI(srv.dbURL))
	if err == nil {
		err = srv.mongoClient.Connect(nil)
	}
	//ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*100)
	//defer cancelFunc()
	if err == nil {
		err = srv.mongoClient.Ping(ctx, nil)
	}
	if err != nil {
		logrus.Errorf("failed to connect to MongoDB on %s: %s", srv.dbURL, err)
		return err
	}
	srv.storeDB = srv.mongoClient.Database(srv.clientID)
	return err
}

// Close disconnects from the DB server
// Call Connect to reconnect.
func (srv *MongoBucketStore) Close() error {
	logrus.Infof("Disconnecting from the database")
	ctx := context.Background()
	//ctx, cf := context.WithTimeout(context.Background(), 10*time.Second)
	err := srv.mongoClient.Disconnect(ctx)
	srv.mongoClient = nil
	//cf()
	return err
}

// NewMongoBucketStore creates a bucket store with the MongoDB backend.
// This is intended for basic key-value storage use.
//
//	clientID is used as the database name
//	dbURL and pass are used to connect to the database
//	useTimeSeries creates timeseries collections for buckets instead of regular buckets
func NewMongoBucketStore(clientID, dbURL, pass string, useTimeSeries bool) *MongoBucketStore {

	srv := &MongoBucketStore{
		clientID:      clientID,
		dbURL:         dbURL,
		dbPass:        pass,
		useTimeSeries: useTimeSeries,
	}
	return srv
}
