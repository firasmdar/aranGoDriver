# aranGoDriver

This project is a golang-driver for [ArangoDB](https://www.arangodb.com/)

Currently implemented:
* connect to DB
* databases
  * List all databases
  * create a database
  * drop a database

## Test

### Test against a fake-in-memory-database:
```
go test
```

### Test with a real database
```
go test -database
```

#### fit tests with a real database

1. Open file [aranGoDriver_test.go](https://github.com/TobiEiss/aranGoDriver/blob/master/aranGoDriver_test.go)
2. Edit following const's:
  * `testUsername` username for database
  * `testPassword` password for database
  * `testDbName` test-name for a not existing database
  * `testDbHost` the host of your database (e.g.: `http://localhost:8529`)

## Usage

### Connect to your ArangoDB

You need a new Session to your database with the hostname as parameter. Then connect with an existing username and a password.
```
session := aranGoDriver.NewAranGoDriverSession("http://localhost:8529")
session.Connect("username", "password")
```

### List all database
You will get all databases as string-slice (`[]string`)
```
list := session.ListDBs()
fmt.Println(list)
```

will print:
`[ _system test testDB]`

### Create a new database
Just name it!
```
session.CreateDB("myNewDatabase")
```

### Drop a database
And now lets drop this database
```
session.DropDB("myNewDatabase")
```
