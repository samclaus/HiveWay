package main

import (
	"errors"

	"github.com/vmihailenco/msgpack/v5"
)

// RegistrationTokenSpec defines user-configurable fields for registration tokens. See
// RegistrationTokenInfo for more information.
type RegistrationTokenSpec struct {
	// The ID of the token IS the token, i.e., what users see/use.
	ID string `gorm:"primaryKey" msgpack:"id"`
	// Role is the role that will be granted to the user who registers with the token.
	Role uint `gorm:"role" msgpack:"role"`
	// Notes is just generic text entered by the admin that created the
	// token. Useful for mentioning who the token is intended  for.
	Notes string `gorm:"notes" msgpack:"notes"`
}

// RegistrationTokenInfo describes a registration token. Tokens are created by admins
// to permit new users to register. Every token is single-use.
type RegistrationTokenInfo struct {
	RegistrationTokenSpec
	// CreatedBy is the ID of the admin that created the token.
	CreatedBy string `gorm:"created_by" msgpack:"created_by"`
}

func listRegistrationTokens(s *Server, u *UserInfo, payload []byte) (any, error) {
	// TODO: only admins can list tokens
	var tokens []RegistrationTokenInfo
	if err := s.Database.Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func createRegistrationToken(s *Server, u *UserInfo, payload []byte) (any, error) {
	var spec RegistrationTokenSpec
	if err := msgpack.Unmarshal(payload, &spec); err != nil {
		// TODO
		return nil, err
	}

	// Add the CreatedBy field to obtain the full info
	info := RegistrationTokenInfo{spec, u.ID}

	if err := s.Database.Create(info).Error; err != nil {
		// TODO
		return nil, err
	}

	return info, nil
}

func deleteRegistrationToken(s *Server, u *UserInfo, payload []byte) (any, error) {
	var id string
	if err := msgpack.Unmarshal(payload, &id); err != nil {
		// TODO
		return nil, err
	}

	if id == "" {
		return nil, errors.New("a non-empty string ID must be supplied")
	}

	if err := s.Database.Delete(&RegistrationTokenInfo{}, "id = ?", id).Error; err != nil {
		// TODO
		return nil, err
	}

	return nil, nil
}
