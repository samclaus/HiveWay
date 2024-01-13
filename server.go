package main

import (
	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Server represents this server as a whole and contains global configuration
// information so request-handling code has a single spot to read it from.
type Server struct {
	// The database where all our persistent information will be stored, i.e.,
	// basically everything. Under the hood, the information will get stored as
	// a file on the disk because we will be using SQLite for now. That may
	// change in the future.
	Database *gorm.DB
}

// NewServer attempts to open the given database file and returns a new Server if
// successful.
func NewServer(dbPath string) (*Server, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "error opening database")
	}

	if err = db.AutoMigrate(&UserInfo{}); err != nil {
		// TODO: close database?
		return nil, errors.Wrap(err, "error migrating user schema")
	}

	return &Server{db}, nil
}
