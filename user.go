package main

func listUsers(s *Server, u *UserInfo, payload []byte) (any, error) {
	var users []UserInfo
	if err := s.Database.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
