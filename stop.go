package main

import (
	"errors"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

type StopInfo struct {
	ID                 string  `gorm:"primaryKey" json:"id" msgpack:"id"`
	Code               string  `gorm:"code" json:"code" msgpack:"code"`
	Name               string  `gorm:"name" json:"name" msgpack:"name"`
	NameTTS            string  `gorm:"name_tts" json:"name_tts,omitempty" msgpack:"name_tts,omitempty"`
	Description        string  `gorm:"description" json:"description" msgpack:"description"`
	Lat                float64 `gorm:"lat" json:"lat" msgpack:"lat"`
	Lng                float64 `gorm:"lng" json:"lng" msgpack:"lng"`
	ZoneID             string  `gorm:"zone_id" json:"zone_id,omitempty" msgpack:"zone_id,omitempty"`
	URL                string  `gorm:"url" json:"url,omitempty" msgpack:"url,omitempty"`
	Type               uint    `gorm:"type" json:"type" msgpack:"type"`
	ParentStation      string  `gorm:"parent_station" json:"parent_station,omitempty" msgpack:"parent_station,omitempty"`
	Timezone           string  `gorm:"timezone" json:"timezone,omitempty" msgpack:"timezone,omitempty"`
	WheelchairBoarding uint    `gorm:"wheelchair_boarding" json:"wheelchair_boarding" msgpack:"wheelchair_boarding"`
	LevelID            string  `gorm:"level_id" json:"level_id,omitempty" msgpack:"level_id,omitempty"`
	PlatformCode       string  `gorm:"platform_code" json:"platform_code,omitempty" msgpack:"platform_code,omitempty"`
}

func createStop(s *Server, u *UserInfo, payload []byte) (any, error) {
	var info StopInfo
	if err := msgpack.Unmarshal(payload, &info); err != nil {
		// TODO
		return nil, err
	}

	if info.ID == "" {
		id, err := uuid.NewRandom()
		if err != nil {
			// TODO
			return nil, err
		}
		info.ID = id.String()
	}

	if err := s.Database.Create(info).Error; err != nil {
		// TODO
		return nil, err
	}

	return info, nil
}

func deleteStop(s *Server, u *UserInfo, payload []byte) (any, error) {
	var id string
	if err := msgpack.Unmarshal(payload, &id); err != nil {
		// TODO
		return nil, err
	}

	if id == "" {
		// TODO
		return nil, errors.New("a non-empty string ID must be supplied")
	}

	if err := s.Database.Delete(&StopInfo{}, "id = ?", id).Error; err != nil {
		// TODO
		return nil, err
	}

	return nil, nil
}
