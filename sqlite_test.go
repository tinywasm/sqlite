package sqlite_test

import (
	"testing"

	"github.com/cdvelop/objectdb/tests"
	"github.com/cdvelop/sqlite"
)

func Test_SqliteMemoryMode(t *testing.T) {

	db := sqlite.NewConnection("", "test_memory", true)
	tests.Run(db, t)

}

func Test_SqlitePersistenMode(t *testing.T) {
	//test.....
	root := "./test_files"
	db_name := "test.db"

	db := sqlite.NewConnection(root, db_name, false)

	tests.Run(db, t)

}
