package config

import (
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func NewInitializedSQLiteDatabase() (*sqlstore.Container, error) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	db, err := sqlstore.New("sqlite3", "file:go_whatsapp.db?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, err
	}

	return db, nil
}
