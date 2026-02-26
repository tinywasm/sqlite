package sqlite

import (
	"github.com/cdvelop/objectdb"
	_ "github.com/mattn/go-sqlite3"
)

// root_folder ej: "./app_files" db_name ej: mi_proyecto.db min 4 caracteres
// mode_memory solo en memoria
func NewConnection(root_folder, db_name string, mode_memory bool) *objectdb.Connection {

	if db_name == "" || len(db_name) < 4 {
		showErrorAndExit("NOMBRE BASE DE DATOS INCORRECTO")
	}

	dba := db{
		rootFolder:   root_folder + "/",
		dataBaseName: db_name,
		modeMemory:   mode_memory,
	}

	db := objectdb.Get(&dba)

	return db
}
