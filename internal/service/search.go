package service

import (
	"context"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/model"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/repo"
)

type SearchService struct {
	repo *repo.SearchRepo
}

func NewSearchService(repo *repo.SearchRepo) *SearchService {
	return &SearchService{repo: repo}
}

func (s *SearchService) Search(ctx context.Context, query string) ([]model.User, error) {
	return s.repo.SearchByName(ctx, query)
}

type FeedResult struct {
	Users []model.User `json:"users"`
	Total int          `json:"total"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
}

func (s *SearchService) Feed(ctx context.Context, page, limit int) (*FeedResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	users, total, err := s.repo.Feed(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	return &FeedResult{
		Users: users,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}
