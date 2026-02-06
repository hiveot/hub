# Bucket Store 

The bucketstore provides simple embedded and non-embedded key-value persistence for Hub services with the focus on simplicity. 
It has a standardized key-value API and can support multiple backends. 

What the bucketstore is not, is a general purpose database. It is intended to be simple to use meet basic storage needs. If you need multiple indexes then this store is not the right choice. 

## Concept 

The bucket store concept provided through the API of the bucket store is:

 open store -> read/write bucket -> iterate using cursor 
 
Where:
* Store is a database instance per client. Client being a service that needs persistence.
* Bucket is a collection of key-value pairs in the store. Supported operations are get (multiple), set (multiple), and delete. 
* Cursor is an iterator in a bucket to iterate to the first, last, next, previous and seek a specific key.

That is all there is to it. No magic.

## Backends

Short description of the supported backends.

Note that the implementation hasn't been optimized for performance and that the default settings are being used. Especially Pebble and BoltDB have many capabilities and tricks to might be able to significantly improve read/write performance.  

### kvbtree

The kvbtree backend is an embedded in-memory store using a btree as a store per client. Data is serialized and persisted to one file per client. Data is periodically written to disk after modifications are made.

This backend is exceptionally fast and the fastest of the available backends for both reading and writing. A read and a write takes less than 1 usec per record, so a speed of a million read/writes per second is possible.  

All the data is kept in memory, so the capacity depends on available memory. Kinda like Redis does. Data is not compressed. When the store is updated, a background process periodically takes a snapshot (shallow copy) and writes it to disk. In writing to disk it first writes to a temporary file and when successful, renames the temporary file. This avoids the risk of data corruption.  

This store is best suited for limited amount of data, based on memory, that is frequently read and updated. The recommended data limit is 100MB. Testing has shows 

### pebble

The pebble backend is cockroachdb's persistence layer. It is all around awesome and probably a bit overkill.

It is also an embedded database which explains is very high speed. While kvbtree is around 5-10 faster, it is still very fast with both reading and writing taking approx 1-2usec per record (see BucketBench_test.go for details.)  

Pebble's data size is pretty much limited to the available disk space. Got 1TB, well you can store 1TB without suffering too much of a performance penalty. (although this is only tested with about 10 million records). Data is compressed so actual disk space is likely to be less.

This store is best suited for large amounts of data. For example, the time series data of the history store.

### bolts - removed as overkill

The boltDB (bbolt implementation) backend is a solid transactional embedded database. 

Its read speed is close to that of pebble. Read speed does tend to suffer for large amount of data, in the order of 1 million records or more.

However, writing is rather slow with about 5msec per write transaction. Write speed can be greatly increase by using SetMultiple. For example, writing 1000 key-value pairs take less than twice as long as writing a single key-value pair. Just as with reading, writing gets noticeable slower when reaching a high number of records.

BBolt's data size is also limited to available disk space. Data isn't compressed.

This store is best suited for compatiblity with other BoltDB databases or tools. 


### mongo - abandoned

The mongoDB backend is not complete. One of the main stumbling blocks is that mongodb only has a forward iterator. In addition, mongodb is a standalone server while the other options are embedded databases. The performance of MongoDB will therefore not be able to compete with the others. 


### redis - not planned 
Redis is not an embedded store and requires external setup and maintenance. It is out of scope for this application.  
That said, it has a well defined interface and superb performance so if a use-case comes up it can be considered.