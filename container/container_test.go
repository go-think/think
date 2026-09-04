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

// Reader is an interface for testing interface-to-implementation binding.
type Reader interface {
	ReadData() string
}

type FileReader struct {
	Path string
}

func (f *FileReader) ReadData() string {
	return "file:" + f.Path
}

func TestContainer_InterfaceBinding_Func(t *testing.T) {
	c := New()

	// Register interface Reader bound to concrete *FileReader via factory func
	c.Singleton[Reader](func() *FileReader {
		return &FileReader{Path: "/etc/config"}
	})

	reader := c.Make[Reader]()
	assert.NotNil(t, reader)
	assert.Equal(t, "file:/etc/config", reader.ReadData())
}

func TestContainer_InterfaceBinding_Object(t *testing.T) {
	c := New()

	// Register interface Reader directly bound to concrete implementation instance
	c.Singleton[Reader](&FileReader{Path: "/app/data"})

	reader := c.Make[Reader]()
	assert.NotNil(t, reader)
	assert.Equal(t, "file:/app/data", reader.ReadData())
}

func TestContainer_TypeSafeAlias(t *testing.T) {
	c := New()

	c.Singleton[*Database](func() *Database {
		return &Database{URL: "postgres://localhost"}
	})

	// 1. Alias using pointer type: c.Alias[*Database]("db")
	c.Alias[*Database]("db")
	resolved1 := c.MakeByName("db")
	assert.NotNil(t, resolved1)
	db1, ok := resolved1.(*Database)
	assert.True(t, ok)
	assert.Equal(t, "postgres://localhost", db1.URL)

	// 2. Also resolvable via generic Make[*Database]("db")
	resolved2 := c.Make[*Database]("db")
	assert.NotNil(t, resolved2)
	assert.Equal(t, "postgres://localhost", resolved2.URL)
}

func TestContainer_NoShortNameCollision(t *testing.T) {
	c := New()

	type ServiceA struct {
		Name string
	}
	type ServiceB struct {
		Name string
	}

	c.Singleton[*ServiceA](func() *ServiceA {
		return &ServiceA{Name: "Alpha"}
	})
	c.Singleton[*ServiceB](func() *ServiceB {
		return &ServiceB{Name: "Beta"}
	})

	// Both should resolve safely by their distinct types without any collision
	a := c.Make[*ServiceA]()
	b := c.Make[*ServiceB]()

	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.Equal(t, "Alpha", a.Name)
	assert.Equal(t, "Beta", b.Name)
}

func TestContainer_Resolve_And_Invoke_Safe(t *testing.T) {
	c := New()

	// 1. Resolve for unregistered service returns explicit error
	type NonExistent struct{}
	val, err := c.Resolve[*NonExistent]()
	assert.Error(t, err)
	assert.Nil(t, val)

	// Make for unregistered service safely returns nil without panic
	safeMake := c.Make[*NonExistent]()
	assert.Nil(t, safeMake)

	// 2. Invoke for non-function returns error instead of panic
	out, err := c.Invoke("not_a_function")
	assert.Error(t, err)
	assert.Nil(t, out)

	// Call for non-function safely returns nil without panic
	safeCall := c.Call(12345)
	assert.Nil(t, safeCall)
}
