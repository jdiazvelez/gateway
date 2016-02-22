package store_test

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	"gateway/config"
	"gateway/store"

	"github.com/ory-am/dockertest"
)

var testStore store.Store

func TestMain(m *testing.M) {
	flag.Parse()

	var postgresStore store.Store
	conf := config.Store{
		Type: "postgres",
	}
	dockertest.BindDockerToLocalhost = "true"
	c, err := dockertest.ConnectToPostgreSQL(60, time.Second, func(url string) bool {
		conf.ConnectionString = url
		postgresStore, _ = store.Configure(conf)
		return postgresStore.(*store.PostgresStore).Ping() == nil
	})
	if err != nil {
		log.Fatalf("Could not connect to database: %s", err)
	}
	defer func() {
		postgresStore.Shutdown()
		c.KillRemove()
	}()
	testStore = postgresStore
	status := m.Run()

	var boltStore store.Store
	file := make([]byte, 8)
	binary.BigEndian.PutUint64(file, uint64(rand.Int63()))
	name := os.TempDir() + string(os.PathSeparator) + hex.EncodeToString(file) + ".db"
	conf = config.Store{
		Type:             "boltdb",
		ConnectionString: name,
	}
	boltStore, err = store.Configure(conf)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		boltStore.Shutdown()
		err := os.Remove(name)
		if err != nil {
			log.Fatal(err)
		}
	}()
	testStore = boltStore
	status = m.Run()

	os.Exit(status)
}

var testJson = []string{
	`{
		"name": {
			"first": "John",
			"last": "Doe"
		},
		"age": 25
	}`,
	`{
		"name": {
			"first": "Jane",
			"last": "Doe"
		},
		"age": 21
	}`,
	`{
		"name": {
			"first": "Joe",
			"last": "Schmo"
		},
		"age": 18
	}`,
}

func setup(t *testing.T) (string, store.Store) {
	err := testStore.Migrate()
	if err != nil {
		t.Fatal(err)
	}
	return "", testStore
}

func teardown(t *testing.T, name string, s store.Store) {
	err := testStore.Clear()
	if err != nil {
		t.Fatal(err)
	}
}

func parse(t *testing.T) []interface{} {
	_json := []interface{}{}
	for _, test := range testJson {
		var parsed interface{}
		err := json.Unmarshal([]byte(test), &parsed)
		if err != nil {
			t.Fatal(err)
		}
		_json = append(_json, parsed)
	}
	return _json
}

func TestConfigure(t *testing.T) {
	name, s := setup(t)
	teardown(t, name, s)
}

func TestCreateCollection(t *testing.T) {
	name, s := setup(t)
	defer teardown(t, name, s)

	collection := &store.Collection{AccountID: 0, Name: "acollection"}
	err := s.CreateCollection(collection)
	if err != nil {
		t.Fatal(err)
	}
	if collection.ID == 0 {
		t.Fatal("failed to create collection")
	}
}

func TestListCollection(t *testing.T) {
	name, s := setup(t)
	defer teardown(t, name, s)

	collection := &store.Collection{AccountID: 0, Name: "acollection"}
	err := s.CreateCollection(collection)
	if err != nil {
		t.Fatal(err)
	}
	if collection.ID == 0 {
		t.Fatal("failed to create collection")
	}
	collections := []*store.Collection{}
	err = s.ListCollection(&store.Collection{AccountID: 0}, &collections)
	if err != nil {
		t.Fatal(err)
	}
	if len(collections) != 1 {
		t.Fatal("there should be 1 collection")
	}
}

func TestShowCollection(t *testing.T) {
	name, s := setup(t)
	defer teardown(t, name, s)

	collection := &store.Collection{AccountID: 0, Name: "acollection"}
	err := s.CreateCollection(collection)
	if err != nil {
		t.Fatal(err)
	}
	if collection.ID == 0 {
		t.Fatal("failed to create collection")
	}

	err = s.ShowCollection(collection)
	if err != nil {
		t.Fatal(err)
	}
	if collection.Name == "" {
		t.Fatal("collection name should be set")
	}
}

func TestUpdateCollection(t *testing.T) {
	name, s := setup(t)
	defer teardown(t, name, s)

	collection := &store.Collection{AccountID: 0, Name: "acollection"}
	err := s.CreateCollection(collection)
	if err != nil {
		t.Fatal(err)
	}
	if collection.ID == 0 {
		t.Fatal("failed to create collection")
	}

	collection.Name = "anewname"
	err = s.UpdateCollection(collection)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteCollection(t *testing.T) {
	name, s := setup(t)
	defer teardown(t, name, s)

	collection := &store.Collection{AccountID: 0, Name: "acollection"}
	err := s.CreateCollection(collection)
	if err != nil {
		t.Fatal(err)
	}
	if collection.ID == 0 {
		t.Fatal("failed to create collection")
	}

	err = s.DeleteCollection(collection)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInsert(t *testing.T) {
	name, s := setup(t)
	defer teardown(t, name, s)

	objects := parse(t)
	objects, err := s.Insert(0, "people", objects[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range objects {
		if object.(map[string]interface{})["$id"] == nil {
			t.Fatal("object $id should be set")
		}
	}
}

func TestInsertArray(t *testing.T) {
	name, s := setup(t)
	defer teardown(t, name, s)

	objects := parse(t)
	objs, err := s.Insert(0, "people", objects)
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range objs {
		if object.(map[string]interface{})["$id"] == nil {
			t.Fatal("object $id should be set")
		}
	}
}

func TestSelectByID(t *testing.T) {
	name, s := setup(t)
	defer teardown(t, name, s)

	objects := parse(t)
	objects, err := s.Insert(0, "people", objects[0])
	if err != nil {
		t.Fatal(err)
	}
	object := objects[0]
	if object.(map[string]interface{})["$id"] == nil {
		t.Fatal("object $id should be set")
	}
	id := object.(map[string]interface{})["$id"].(uint64)
	object, err = s.SelectByID(0, "people", id)
	if err != nil {
		t.Fatal(err)
	}
	if object.(map[string]interface{})["$id"] == nil {
		t.Fatal("object $id should be set")
	}
}

func TestUpdateByID(t *testing.T) {
	name, s := setup(t)
	defer teardown(t, name, s)

	objects := parse(t)
	objects, err := s.Insert(0, "people", objects[0])
	if err != nil {
		t.Fatal(err)
	}
	object := objects[0]
	if object.(map[string]interface{})["$id"] == nil {
		t.Fatal("object $id should be set")
	}
	_json := object.(map[string]interface{})
	id := _json["$id"].(uint64)
	_json["age"] = 26
	object, err = s.UpdateByID(0, "people", id, object)
	if err != nil {
		t.Fatal(err)
	}
	if object.(map[string]interface{})["$id"] == nil {
		t.Fatal("object $id should be set")
	}
}

func TestDeleteByID(t *testing.T) {
	name, s := setup(t)
	defer teardown(t, name, s)

	objects := parse(t)
	objects, err := s.Insert(0, "people", objects[0])
	if err != nil {
		t.Fatal(err)
	}
	object := objects[0]
	if object.(map[string]interface{})["$id"] == nil {
		t.Fatal("object $id should be set")
	}
	id := object.(map[string]interface{})["$id"].(uint64)
	object, err = s.DeleteByID(0, "people", id)
	if err != nil {
		t.Fatal(err)
	}
	if object.(map[string]interface{})["$id"] == nil {
		t.Fatal("object $id should be set")
	}
}

func TestDeleteBulk(t *testing.T) {
	name, s := setup(t)
	defer teardown(t, name, s)

	objects := parse(t)
	objs, err := s.Insert(0, "people", objects)
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range objs {
		if object.(map[string]interface{})["$id"] == nil {
			t.Fatal("object $id should be set")
		}
	}

	objs, err = s.Delete(0, "people", "age >= $1", 18)
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 3 {
		t.Fatal("there should be 3 objects")
	}
	for _, object := range objs {
		if object.(map[string]interface{})["$id"] == nil {
			t.Fatal("object $id should be set")
		}
	}
}

func TestSelect(t *testing.T) {
	name, s := setup(t)
	defer teardown(t, name, s)

	objects := parse(t)
	for _, obj := range objects {
		objects, err := s.Insert(0, "people", obj)
		if err != nil {
			t.Fatal(err)
		}
		object := objects[0]
		if object.(map[string]interface{})["$id"] == nil {
			t.Fatal("object $id should be set")
		}
	}
	objects, err := s.Select(0, "people", "age >= 21 order numeric(age) asc")
	t.Log(objects)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 2 {
		t.Fatal("there should be 2 objects")
	}
	for _, object := range objects {
		if object.(map[string]interface{})["$id"] == nil {
			t.Fatal("object $id should be set")
		}
	}
	last := int64(0)
	for _, object := range objects {
		age := object.(map[string]interface{})["age"].(float64)
		if int64(age) < last {
			t.Fatal("objects should be sorted")
		}
		last = int64(age)
	}

	objects, err = s.Select(0, "people", "true")
	t.Log(objects)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 3 {
		t.Fatal("there should be 3 objects")
	}
}
