package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/model"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/repo"
	"github.com/google/uuid"
)

type UserService struct {
	repo *repo.UserRepo
}

func NewUserService(repo *repo.UserRepo) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Create(ctx context.Context, name, email string) (*model.User, error) {
	exists, err := s.repo.EmailExists(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("email already registered")
	}

	user := &model.User{
		ID:        uuid.NewString(),
		Name:      name,
		Email:     email,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Insert(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id string) (*model.User, error) {
	return s.repo.GetByID(ctx, id)
}
