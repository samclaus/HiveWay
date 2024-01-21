package main

// Config encapsulates all user-configurable properties of a HiveWay system.
type Config struct {
	// File path to the main SQLite database. If this path is not absolute, it
	// will be resolved relative to the location of the config file.
	DatabasePath string `toml:"database_path"`

	// Whenever a user registers, they need to provide a registration token
	// that has been prepared in advance by an administrator. In order for
	// the FIRST administrator to make THEIR account, they must provide the
	// token that is specified here. The user that registers with this token
	// will have special root privileges, such as creating registration tokens
	// that will allow new ADMINS to register.
	//
	// When registering using this token, the system will first check if there
	// is an existing root account. If there is, the registration attempt will
	// be rejected. Otherwise, the account will be created with the root rank.
	RootRegistrationToken string `toml:"root_registration_token"`
}
