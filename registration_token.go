package main

import (
	"errors"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

// RegistrationTokenSpec defines user-configurable fields for registration tokens. See
// RegistrationTokenInfo for more information.
type RegistrationTokenSpec struct {
	// The ID of the token IS the token, i.e., what users see/use.
	ID string `gorm:"primaryKey" msgpack:"id"`
	// Role is the role that will be granted to the user who registers with the token.
	Role uint `gorm:"role" msgpack:"role"`
	// Name of the person this token is for.
	Name string `gorm:"name" msgpack:"name"`
	// Notes is just optional generic text entered by the admin that created the
	// token.
	Notes string `gorm:"notes" msgpack:"notes"`
}

// RegistrationTokenInfo describes a registration token. Tokens are created by admins
// to permit new users to register. Every token is single-use.
type RegistrationTokenInfo struct {
	RegistrationTokenSpec
	// CreatedAt is a timestamp of when the token was created.
	CreatedAt uint64 `gorm:"created_at" msgpack:"created_at"`
	// CreatedBy is the ID of the admin that created the token.
	CreatedBy string `gorm:"created_by" msgpack:"created_by"`
}

func listRegistrationTokens(s *Server, u *UserInfo, payload []byte) (any, error) {
	// TODO: improve error
	if u.Role == 0 {
		return nil, &ErrorWithCode{
			Code:    "role-too-low",
			Message: "only admins have permission to view registration tokens",
		}
	}

	var tokens []RegistrationTokenInfo
	if err := s.Database.Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func createRegistrationToken(s *Server, u *UserInfo, payload []byte) (any, error) {
	// TODO: improve error
	if u.Role == 0 {
		return nil, &ErrorWithCode{
			Code:    "role-too-low",
			Message: "only admins have permission to create registration tokens",
		}
	}

	var spec RegistrationTokenSpec
	if err := msgpack.Unmarshal(payload, &spec); err != nil {
		// TODO
		return nil, err
	}

	// Add the CreatedBy field to obtain the full info
	info := RegistrationTokenInfo{spec, uint64(time.Now().UnixMilli()), u.ID}

	if err := s.Database.Create(info).Error; err != nil {
		// TODO
		return nil, err
	}

	return info, nil
}

func deleteRegistrationToken(s *Server, u *UserInfo, payload []byte) (any, error) {
	// TODO: improve error
	if u.Role == 0 {
		return nil, &ErrorWithCode{
			Code:    "role-too-low",
			Message: "only admins have permission to delete registration tokens",
		}
	}

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
