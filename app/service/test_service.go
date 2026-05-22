package service

import (
	"house-hunt/repository"
)

type TestService struct {
	Repo *repository.TestRepository
}

func NewTestService(repo *repository.TestRepository) *TestService {
	return &TestService{Repo: repo}
}

func (s *TestService) PingDB() error {
	err := s.Repo.PingDB()
	if err != nil {
		return err
	}
	return nil
}
