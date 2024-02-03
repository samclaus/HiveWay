package main

import (
	"errors"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

type PathSpec struct {
	Line        bool                `gorm:"line" json:"line" msgpack:"line"`
	Coords      *msgpack.RawMessage `gorm:"coords" json:"coords" msgpack:"coords"`
	Description string              `gorm:"description" json:"description" msgpack:"description"`
	Styles      *msgpack.RawMessage `gorm:"styles" json:"styles" msgpack:"styles"`
}

type PathInfo struct {
	PathSpec
	ID string `gorm:"primaryKey" json:"id" msgpack:"id"`
}

func createPath(s *Server, u *UserInfo, payload []byte) (any, error) {
	var spec PathSpec
	if err := msgpack.Unmarshal(payload, &spec); err != nil {
		// TODO
		return nil, err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		// TODO
		return nil, err
	}

	info := PathInfo{spec, id.String()}

	if err := s.Database.Create(info).Error; err != nil {
		// TODO
		return nil, err
	}

	return info, nil
}

func deletePath(s *Server, u *UserInfo, payload []byte) (any, error) {
	var id string
	if err := msgpack.Unmarshal(payload, &id); err != nil {
		// TODO
		return nil, err
	}

	if id == "" {
		// TODO
		return nil, errors.New("a non-empty string ID must be supplied")
	}

	if err := s.Database.Delete(&PathInfo{}, "id = ?", id).Error; err != nil {
		// TODO
		return nil, err
	}

	return nil, nil
}
