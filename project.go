package main

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
	"gorm.io/gorm/clause"
)

// ProjectSpec defines user-configurable fields for registration tokens. See
// ProjectInfo for more information.
type ProjectSpec struct {
	Name string `gorm:"name" msgpack:"name"`
	Desc string `gorm:"desc" msgpack:"desc"`
}

// ProjectInfo describes a registration token. Tokens are created by admins
// to permit new users to register. Every token is single-use.
type ProjectInfo struct {
	ProjectSpec
	ID string `gorm:"primaryKey" msgpack:"id"`
	// CreatedAt is a timestamp of when the project was created.
	CreatedAt uint64 `gorm:"created_at" msgpack:"created_at"`
	// CreatedBy is the ID of the user that created the project.
	CreatedBy string `gorm:"created_by" msgpack:"created_by"`
}

type ProjectFeatures struct {
	Stops   []StopInfo   `json:"stops" msgpack:"stops"`
	Paths   []PathInfo   `json:"paths" msgpack:"paths"`
	Circles []CircleInfo `json:"circles" msgpack:"circles"`
}

func listProjects(s *Server, u *UserInfo, payload []byte) (any, error) {
	var projects []ProjectInfo
	if err := s.Database.Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func createProject(s *Server, u *UserInfo, payload []byte) (any, error) {
	var spec ProjectSpec
	if err := msgpack.Unmarshal(payload, &spec); err != nil {
		// TODO
		return nil, err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		// TODO
		return nil, err
	}

	info := ProjectInfo{spec, id.String(), uint64(time.Now().UnixMilli()), u.ID}

	if err := s.Database.Create(info).Error; err != nil {
		// TODO
		return nil, err
	}

	return info, nil
}

func modifyProjectMetadata(s *Server, u *UserInfo, payload []byte) (any, error) {
	// They might try to modified 'created_at' and other fields they are not allowed
	// to modify
	var untrustedChanges map[string]any
	if err := msgpack.Unmarshal(payload, &untrustedChanges); err != nil {
		// TODO
		return nil, err
	}

	id, _ := untrustedChanges["id"].(string)
	if id == "" {
		// TODO
		return nil, errors.New("a non-empty string ID must be supplied")
	}

	changes := map[string]any{}

	if name, ok := untrustedChanges["name"].(string); ok {
		changes["name"] = name
	}
	if desc, ok := untrustedChanges["desc"].(string); ok {
		changes["desc"] = desc
	}

	proj := ProjectInfo{ID: id}
	if err := s.Database.Model(&proj).Clauses(clause.Returning{}).Updates(changes).Error; err != nil {
		// TODO
		return nil, err
	}

	return proj, nil
}

func deleteProject(s *Server, u *UserInfo, payload []byte) (any, error) {
	var id string
	if err := msgpack.Unmarshal(payload, &id); err != nil {
		// TODO
		return nil, err
	}

	if id == "" {
		// TODO
		return nil, errors.New("a non-empty string ID must be supplied")
	}

	if err := s.Database.Delete(&ProjectInfo{}, "id = ?", id).Error; err != nil {
		// TODO
		return nil, err
	}

	return nil, nil
}

func listProjectFeatures(s *Server, u *UserInfo, payload []byte) (any, error) {
	var features ProjectFeatures

	if err := s.Database.Find(&features.Stops).Error; err != nil {
		return nil, err // TODO
	}
	if err := s.Database.Find(&features.Paths).Error; err != nil {
		return nil, err // TODO
	}
	if err := s.Database.Find(&features.Circles).Error; err != nil {
		return nil, err // TODO
	}

	return features, nil
}
