package main

import (
	"fmt"
	"github.com/cockroachdb/pebble"
	"os"
)

// temporary - testing of pebble DB iteration
// some of these might be useful to build into the bucketstore
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Missing a pebble database path")
		os.Exit(1)
	}
	dbPath := os.Args[1]
	_, err := os.Stat(dbPath)
	if err != nil {
		fmt.Println("Not a valid path")
		os.Exit(1)
	}

	options := &pebble.Options{}
	inDB, err := pebble.Open(dbPath, options)
	if err != nil {
		fmt.Println("Error opening database: ", err.Error())
		os.Exit(1)
	}
	defer inDB.Close()

	// only one input
	if len(os.Args) == 2 {
		metrics(inDB)
		//iterkeys(inDB)
	} else if len(os.Args) == 3 {
		query(inDB, os.Args[2])
		//firstlast(inDB, os.Args[2])
		//copyTo(inDB,os.Args[2]
	}
}

func metrics(db *pebble.DB) {
	m := db.Metrics()
	fmt.Println("---")
	fmt.Printf("diskspace: %d\n", m.DiskSpaceUsage())
	fmt.Println(m.String())
	fmt.Println("---")
}

// print all keys in the db
func iterkeys(db *pebble.DB) error {
	opts := pebble.IterOptions{}
	iter, err := db.NewIter(&opts)
	if err != nil {
		return err
	}
	i := 0
	for iter.First(); iter.Valid(); iter.Next() {
		fmt.Printf("key=%q value=%-20.20q\n", iter.Key(), iter.Value())
		i++
	}
	fmt.Printf("Found '%d' keys\n", i)
	return nil
}

// copy the database keys to the destination
func copyTo(inDB *pebble.DB, dest string) {
	_, err := os.Stat(dest)
	if err == nil {
		// todo use -f
		fmt.Println("Value database already exists. Not copying keys.")
		return
	}
	options := &pebble.Options{}
	outDB, err := pebble.Open(dest, options)
	if err != nil {
		// todo use -f
		fmt.Println("Error creating output database:", err.Error())
		return
	}
	defer outDB.Close()
	copykeys(inDB, outDB)
}

// copy the database keys to another database
func copykeys(dbIn *pebble.DB, dbOut *pebble.DB) error {
	opts := pebble.IterOptions{}
	iter, err := dbIn.NewIter(&opts)
	if err != nil {
		return err
	}
	writeOpts := pebble.WriteOptions{}
	i := 0
	for iter.First(); iter.Valid(); iter.Next() {
		k := iter.Key()
		v, err := iter.ValueAndErr()
		if err == nil {
			err = dbOut.Set(k, v, &writeOpts)
		}
		if err != nil {
			fmt.Printf("Copy error: %s", err.Error())
			return err
		}
		i++
	}
	_ = dbOut.Flush()
	fmt.Printf("Copied '%d' keys\n", i)
	return nil
}

// query all keys with the given prefix (bucket name/thingID)
func query(dbIn *pebble.DB, prefix string) {
	opts := pebble.IterOptions{
		LowerBound: []byte(prefix + "$"),
		UpperBound: []byte(prefix + "%"), // this key never exists
	}
	iter, err := dbIn.NewIter(&opts)
	if err != nil {
		fmt.Println("Key not found in DB: ", prefix)
		return
	}
	i := 0
	valid := iter.SeekGE([]byte(prefix))
	for valid {
		key := iter.Key()
		//if !strings.HasPrefix(string(key), prefix) {
		//	break
		//}
		fmt.Printf("key=%s value=%-20.20s\n", key, iter.Value())
		valid = iter.Next()
		i++
	}
	fmt.Printf("Found '%d' keys\n", i)
}

// return the last key with the given prefix (bucket name/thingID)
func firstlast(dbIn *pebble.DB, prefix string) {
	opts := pebble.IterOptions{
		LowerBound: []byte(prefix + "$"),
		UpperBound: []byte(prefix + "%"), // this key never exists
	}
	iter, err := dbIn.NewIter(&opts)
	if err != nil {
		fmt.Println("Key not found in DB: ", prefix)
		return
	}

	valid := iter.First()
	if valid {
		fmt.Printf("Frst: key=%s value=%-20.20s\n", iter.Key(), iter.Value())
	}
	valid = iter.Last()
	if valid {
		fmt.Printf("Last: key=%s value=%-20.20s\n", iter.Key(), iter.Value())
	}
}
