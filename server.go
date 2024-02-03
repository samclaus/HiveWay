package main

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type RequestHandler = func(*Server, *UserInfo, []byte) (any, error)

// Server represents this server as a whole and contains global configuration
// information so request-handling code has a single spot to read it from.
type Server struct {
	// The database where all our persistent information will be stored, i.e.,
	// basically everything. Under the hood, the information will get stored as
	// a file on the disk because we will be using SQLite for now. That may
	// change in the future.
	Database *gorm.DB

	// Registration token for bootstrapping the system with the first/root user.
	RootRegToken string

	// Handlers for various request types, like "user:list" or "registration_token:delete".
	RequestHandlers map[string]RequestHandler
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

	if err = db.AutoMigrate(
		&UserInfo{},
		&RegistrationTokenInfo{},
		&ProjectInfo{},
		&StopInfo{},
		&PathInfo{},
		&CircleInfo{},
	); err != nil {
		// TODO: close database?
		return nil, errors.Wrap(err, "error migrating database schema")
	}

	return &Server{
		Database:     db,
		RootRegToken: cfg.RootRegistrationToken,
		RequestHandlers: map[string]RequestHandler{
			"registration_token:list":   listRegistrationTokens,
			"registration_token:create": createRegistrationToken,
			"registration_token:delete": deleteRegistrationToken,
			"user:list":                 listUsers,
			"user:delete":               deleteUser,
			"project:list":              listProjects,
			"project:create":            createProject,
			"project:modify":            modifyProjectMetadata,
			"project:delete":            deleteProject,
			"project:list_features":     listProjectFeatures,
			"stop:create":               createStop,
			"stop:delete":               deleteStop,
			"path:create":               createPath,
			"path:delete":               deletePath,
			"circle:create":             createCircle,
			"circle:delete":             deleteCircle,
		},
	}, nil
}
