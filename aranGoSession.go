package aranGoDriver

import (
	"errors"
	"net/http"

	"github.com/TobiEiss/aranGoDriver/aranGoConnection"
	"github.com/TobiEiss/aranGoDriver/models"
)

// AranGoSession represent to Session
type AranGoSession struct {
	arangoCon *aranGoConnection.AranGoConnection
}

const urlAuth = "/_open/auth"
const urlDatabase = "/_api/database"
const urlCollection = "/_api/collection"
const urlDocument = "/_api/document"
const urlCursor = "/_api/cursor"

const systemDB = "_system"
const migrationColl = "migrations"

// NewAranGoDriverSession creates a new instance of a AranGoDriver-Session.
// Need a host (e.g. "http://localhost:8529/")
func NewAranGoDriverSession(host string) *AranGoSession {
	return &AranGoSession{aranGoConnection.NewAranGoConnection(host)}
}

// Connect to arangoDB
func (session *AranGoSession) Connect(username string, password string) error {
	credentials := models.Credentials{}
	credentials.Username = username
	credentials.Password = password

	var resultMap map[string]string
	err := session.arangoCon.Query(&resultMap, http.MethodPost, urlAuth, credentials)

	if err == nil {
		session.arangoCon.SetJwtKey(resultMap["jwt"])
	}
	return err
}

// ListDBs lists all db's
func (session *AranGoSession) ListDBs() ([]string, error) {
	var databaseWrapper struct {
		Databases []string `json:"result,omitempty"`
	}
	err := session.arangoCon.Query(&databaseWrapper, http.MethodGet, urlDatabase, nil)

	return databaseWrapper.Databases, err
}

// CreateDB creates a new db
func (session *AranGoSession) CreateDB(dbname string) error {
	body := make(map[string]string)
	body["name"] = dbname
	var result interface{}
	err := session.arangoCon.Query(&result, http.MethodPost, urlDatabase, body)
	return err
}

// DropDB drop a database
func (session *AranGoSession) DropDB(dbname string) error {
	_, _, err := session.arangoCon.Delete(urlDatabase + "/" + dbname)
	return err
}

// CreateCollection creates a collection
func (session *AranGoSession) CreateCollection(dbname string, collectionName string) error {
	body := make(map[string]string)
	body["name"] = collectionName
	var result interface{}
	err := session.arangoCon.Query(&result, http.MethodPost, "/_db/"+dbname+urlCollection, body)
	return err
}

// CreateEdgeCollection creates a edge to DB
func (session *AranGoSession) CreateEdgeCollection(dbname string, edgeName string) error {
	body := make(map[string]interface{})
	body["name"] = edgeName
	body["type"] = 3
	var result interface{}
	err := session.arangoCon.Query(&result, http.MethodPost, "/_db/"+dbname+urlCollection, body)
	return err
}

func (session *AranGoSession) CreateEdgeDocument(dbname string, edgeName string, from string, to string) (models.ArangoID, error) {
	body := make(map[string]interface{})
	body["_from"] = from
	body["_to"] = to
	var aranggoID models.ArangoID
	err := session.arangoCon.Query(&aranggoID, http.MethodPost, "/_db/"+dbname+urlDocument+"/"+edgeName, body)
	return aranggoID, err
}

func (session *AranGoSession) ListCollections(dbname string) (map[string]interface{}, error) {
	var collections map[string]interface{}
	err := session.arangoCon.Query(&collections, http.MethodGet, "/_db/"+dbname+urlCollection, nil)

	return collections, err
}

// DropCollection deletes a collection
func (session *AranGoSession) DropCollection(dbname string, collectionName string) error {
	_, _, err := session.arangoCon.Delete("/_db/" + dbname + urlCollection + "/" + collectionName)
	return err
}

// TruncateCollection truncate collections
func (session *AranGoSession) TruncateCollection(dbname string, collectionName string) error {
	_, _, err := session.arangoCon.Put("/_db/"+dbname+urlCollection+"/"+collectionName+"/truncate", "")
	return err
}

// CreateDocument creates a document in a collection in a database
func (session *AranGoSession) CreateDocument(dbname string, collectionName string, object interface{}) (models.ArangoID, error) {
	var aranggoID models.ArangoID
	err := session.arangoCon.Query(&aranggoID, http.MethodPost, "/_db/"+dbname+urlDocument+"/"+collectionName, object)
	return aranggoID, err
}

// AqlQuery send a query
func (session *AranGoSession) AqlQuery(typ interface{}, dbname string, query string, count bool, batchSize int) error {
	// build request
	requestBody := make(map[string]interface{})
	requestBody["query"] = query
	requestBody["count"] = count
	requestBody["batchSize"] = batchSize

	var result struct {
		Error  bool        `json:"error"`
		Result interface{} `json:"result"`
	}
	result.Result = typ
	err := session.arangoCon.Query(&result, http.MethodPost, "/_db/"+dbname+urlCursor, requestBody)
	if err != nil {
		return err
	}

	if result.Error {
		return errors.New("an error occured")
	}

	return err
}

// GetCollectionByID search collection by id
func (session *AranGoSession) GetCollectionByID(dbname string, id string) (map[string]interface{}, error) {
	var collection map[string]interface{}
	err := session.arangoCon.Query(&collection, http.MethodGet, "/_db/"+dbname+urlDocument+"/"+id, nil)

	return collection, err
}

// UpdateDocument updates an Object
func (session *AranGoSession) UpdateDocument(dbname string, id string, object interface{}) error {
	_, _, err := session.arangoCon.Patch("/_db/"+dbname+urlDocument+"/"+id, object)
	return err
}

// UpdateJSONDocument update a json
func (session *AranGoSession) UpdateJSONDocument(dbname string, id string, jsonObj string) error {
	_, _, err := session.arangoCon.PatchJSON("/_db/"+dbname+urlDocument+"/"+id, []byte(jsonObj))
	return err
}

// Migrate migrates a migration
func (session *AranGoSession) Migrate(migrations ...Migration) error {
	session.CreateCollection(systemDB, migrationColl)

	// helper function
	findMigration := func(name string) (Migration, bool) {
		migrations := []Migration{}
		query := "FOR migration IN " + migrationColl + " FILTER migration.name == '" + name + "' RETURN migration"
		err := session.AqlQuery(&migrations, systemDB, query, true, 1)
		return migrations[0], err == nil
	}

	// iterate all migrations
	for _, mig := range migrations {
		migration, successfully := findMigration(mig.Name)
		if successfully {
			if migration.Status != Finished {
				mig.Handle(session)
				mig.Status = Finished
				session.UpdateDocument(systemDB, mig.ArangoID.ID, mig)
			}
		} else {
			mig.Status = Started
			arangoID, _ := session.CreateDocument(systemDB, migrationColl, mig)
			mig.Handle(session)
			mig.Status = Finished
			session.UpdateDocument(systemDB, arangoID.ID, mig)
		}
	}
	return nil
}
