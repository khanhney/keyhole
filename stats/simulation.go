package stats

import (
	"bytes"
	"log"
	"math"
	"math/rand"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var CollectionName = "keyhole"

// MongoConn -
type MongoConn struct {
	uri    string
	ssl    bool
	sslCA  string
	dbName string
	tps    int
}

// New - Constructor
func New(uri string, ssl bool, sslCA string, dbName string, tps int) MongoConn {
	m := MongoConn{uri, ssl, sslCA, dbName, tps}
	return m
}

// PopulateData - Insert docs to evaluate performance/bandwidth
func (m MongoConn) PopulateData() {
	rand.Seed(time.Now().Unix())
	var buffer bytes.Buffer
	for i := 0; i < 4096/len("simagix."); i++ {
		buffer.WriteString("simagix.")
	}
	s := 0
	batchSize := 20
	if m.tps < batchSize {
		batchSize = m.tps
	}
	for s < 60 {
		s++
		session, err := GetSession(m.uri, m.ssl, m.sslCA)
		if err == nil {
			session.SetMode(mgo.Monotonic, true)
			c := session.DB(m.dbName).C(CollectionName)

			bt := time.Now()
			bulk := c.Bulk()

			for i := 0; i < m.tps; i += batchSize {
				var contentArray []interface{}
				for n := 0; n < batchSize; n++ {
					contentArray = append(contentArray, bson.M{"buffer": buffer.String(), "n": rand.Intn(1000), "ts": time.Now()})
				}
				bulk.Insert(contentArray...)
				_, err := bulk.Run()
				if err != nil {
					log.Println(err)
					session.Close()
					break
				}
			}

			t := time.Now()
			elapsed := t.Sub(bt)
			if elapsed.Seconds() > time.Second.Seconds() {
				x := math.Floor(elapsed.Seconds())
				s += int(x)
				elapsed = time.Duration(elapsed.Seconds() - x)
			}
			et := time.Second.Seconds() - elapsed.Seconds()
			time.Sleep(time.Duration(et))
			session.Close()
		} else {
			time.Sleep(time.Second)
		}
	}
}

// Simulate - Simulate CRUD for load tests
func (m MongoConn) Simulate(duration int) {
	var buffer bytes.Buffer
	for i := 0; i < 4096/len("simagix."); i++ {
		buffer.WriteString("simagix.")
	}

	result := bson.M{}
	results := []bson.M{}
	change := bson.M{"$set": bson.M{"year": 1989}}
	isBurst := false
	burstBegin := time.NewTimer(2 * time.Minute)
	go func() {
		<-burstBegin.C
		isBurst = true
	}()
	burstEnd := time.NewTimer(time.Duration(duration-2) * time.Minute)
	go func() {
		<-burstEnd.C
		isBurst = false
	}()

	for {
		msec := 5
		if isBurst {
			msec = 2
		}
		session, err := GetSession(m.uri, m.ssl, m.sslCA)
		if err == nil {
			session.SetMode(mgo.Monotonic, true)
			c := session.DB(m.dbName).C(CollectionName)
			for i := 0; i < 500; i++ {
				id := bson.NewObjectId()
				_ = c.Insert(bson.M{"_id": id, "buffer": buffer.String(), "n": rand.Intn(1000), "ts": time.Now()})
				time.Sleep(time.Duration(rand.Intn(msec)) * time.Millisecond)
				_ = c.Find(bson.M{"_id": id}).One(&result)
				time.Sleep(time.Duration(rand.Intn(msec)) * time.Millisecond)
				_ = c.Update(bson.M{"_id": id}, change)
				time.Sleep(time.Duration(rand.Intn(msec)) * time.Millisecond)
				_ = c.Remove(bson.M{"_id": id})
				time.Sleep(time.Duration(rand.Intn(msec)) * time.Millisecond)
				_ = c.Find(nil).Limit(10).All(&results)
				time.Sleep(time.Millisecond)
			}
			session.Close()
		}
	}
}

// Cleanup - Drop the temp database
func (m MongoConn) Cleanup() {
	log.Println("cleanup", m.uri)
	session, err := GetSession(m.uri, m.ssl, m.sslCA)
	if err != nil {
		panic(err)
	}
	defer session.Close()
	time.Sleep(2 * time.Second)
	log.Println("dropping collection", m.dbName, CollectionName)
	session.DB(m.dbName).C(CollectionName).DropCollection()
	log.Println("dropping database", m.dbName)
	session.DB(m.dbName).DropDatabase()
}