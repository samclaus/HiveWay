package main

import (
	"errors"

	"github.com/vmihailenco/msgpack/v5"
)

func listUsers(s *Server, u *UserInfo, payload []byte) (any, error) {
	var users []UserInfo
	if err := s.Database.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func deleteUser(s *Server, u *UserInfo, payload []byte) (any, error) {
	// TODO: improve error
	if u.Role == 0 {
		return nil, &ErrorWithCode{
			Code:    "role-too-low",
			Message: "only admins have permission to delete users",
		}
	}

	var id string
	if err := msgpack.Unmarshal(payload, &id); err != nil {
		// TODO
		return nil, err
	}

	if id == "" {
		// TODO
		return nil, errors.New("a non-empty string ID must be supplied")
	}

	var tu UserInfo

	if err := s.Database.Take(&tu, &UserInfo{ID: id}).Error; err != nil {
		// TODO: user might not exist, and handle errors properly
		return nil, err
	}

	if tu.Role >= u.Role {
		return nil, &ErrorWithCode{
			Code:    "role-too-low",
			Message: "you may not delete this user because they are of equal or higher role",
		}
	}

	if err := s.Database.Delete(&UserInfo{}, "id = ?", id).Error; err != nil {
		// TODO
		return nil, err
	}

	return nil, nil
}
