package main

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
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

	// Registration token for bootstrapping the system. See config file documentation.
	BootstrapRegToken string
}

// NewServer attempts to open the given configuration file and initialize a server
// instance from the parameters.
func NewServer(cfgPath string) (*Server, error) {
	cfgFile, err := os.Open(cfgPath)
	if err != nil {
		return nil, errors.Wrap(err, "error reading configuration file")
	}

	var cfg Config
	_, err = toml.NewDecoder(cfgFile).Decode(&cfg)
	cfgFile.Close() // TODO: warn if failed to close file

	if err != nil {
		return nil, errors.Wrap(err, "error parsing configuration file as TOML")
	}

	// Resolve the database path relative to the config path
	if !filepath.IsAbs(cfg.DatabasePath) {
		cfg.DatabasePath = filepath.Join(filepath.Dir(cfgPath), cfg.DatabasePath)
	}

	db, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{})
	if err != nil {
		return nil, errors.Wrapf(err, "error opening database [%s]", cfg.DatabasePath)
	}

	if err = db.AutoMigrate(&UserInfo{}); err != nil {
		// TODO: close database?
		return nil, errors.Wrap(err, "error migrating user table schema")
	}

	return &Server{db, cfg.BootstrapRegistrationToken}, nil
}
