package container

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

type Database struct {
	URL string
}

type UserService struct {
	DB *Database
}

func TestContainerBind(t *testing.T) {
	c := New()

	counter := 0
	c.Bind[*Database](func() *Database {
		counter++
		return &Database{URL: "localhost"}
	}, "db")

	db1 := c.Make[*Database]("db")
	db2 := c.Make[*Database]("db")

	assert.NotNil(t, db1)
	assert.NotNil(t, db2)
	assert.Equal(t, 2, counter, "Bind should create a new instance every time")
	assert.NotSame(t, db1, db2)
}

func TestContainerSingleton(t *testing.T) {
	c := New()

	counter := 0
	c.Singleton[*Database](func() *Database {
		counter++
		return &Database{URL: "localhost"}
	})

	db1 := c.Make[*Database]()
	db2 := c.Make[*Database]()

	assert.NotNil(t, db1)
	assert.NotNil(t, db2)
	assert.Equal(t, 1, counter, "Singleton should only create one instance")
	assert.Same(t, db1, db2)
}

func TestContainerInstance(t *testing.T) {
	c := New()

	db := &Database{URL: "localhost"}
	c.Instance[*Database](db)

	db1 := c.Make[*Database]()
	assert.Same(t, db, db1)
}

func TestContainerScoped(t *testing.T) {
	c := New()

	counter := 0
	c.Scoped[*Database](func() *Database {
		counter++
		return &Database{URL: "scoped_host"}
	})

	db1 := c.Make[*Database]()
	db2 := c.Make[*Database]()
	assert.Same(t, db1, db2)
	assert.Equal(t, 1, counter)

	c.FlushScoped()
	db3 := c.Make[*Database]()
	assert.NotSame(t, db1, db3)
	assert.Equal(t, 2, counter)
}

func TestContainerTags(t *testing.T) {
	c := New()

	c.Instance[*Database](&Database{URL: "r1"}, "rep1")
	c.Instance[*Database](&Database{URL: "r2"}, "rep2")
	c.Tag("rep1", "reports")
	c.Tag("rep2", "reports")

	tagged := c.Tagged[*Database]("reports")
	assert.Len(t, tagged, 2)
	assert.Equal(t, "r1", tagged[0].URL)
	assert.Equal(t, "r2", tagged[1].URL)
}

func TestContainerCallInjection(t *testing.T) {
	c := New()

	c.Instance[*Database](&Database{URL: "localhost"}, "Database")

	results := c.Call(func(db *Database) string {
		if db == nil {
			return "fail"
		}
		return db.URL
	})

	assert.Len(t, results, 1)
	assert.Equal(t, "localhost", results[0])
}
