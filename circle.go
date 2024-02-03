package main

import (
	"errors"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

type CircleSpec struct {
	Center       *msgpack.RawMessage `gorm:"center" json:"center" msgpack:"center"`
	RadiusMeters uint                `gorm:"radius_meters" json:"radius_meters" msgpack:"radius_meters"`
	Description  string              `gorm:"description" json:"description" msgpack:"description"`
	Styles       *msgpack.RawMessage `gorm:"styles" json:"styles" msgpack:"styles"`
}

type CircleInfo struct {
	CircleSpec
	ID string `gorm:"primaryKey" json:"id" msgpack:"id"`
}

func createCircle(s *Server, u *UserInfo, payload []byte) (any, error) {
	var spec CircleSpec
	if err := msgpack.Unmarshal(payload, &spec); err != nil {
		// TODO
		return nil, err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		// TODO
		return nil, err
	}

	info := CircleInfo{spec, id.String()}

	if err := s.Database.Create(info).Error; err != nil {
		// TODO
		return nil, err
	}

	return info, nil
}

func deleteCircle(s *Server, u *UserInfo, payload []byte) (any, error) {
	var id string
	if err := msgpack.Unmarshal(payload, &id); err != nil {
		// TODO
		return nil, err
	}

	if id == "" {
		// TODO
		return nil, errors.New("a non-empty string ID must be supplied")
	}

	if err := s.Database.Delete(&CircleInfo{}, "id = ?", id).Error; err != nil {
		// TODO
		return nil, err
	}

	return nil, nil
}
