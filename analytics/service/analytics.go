package service

import "analytics/repository"

type AnalyticsService struct{}

func NewAnalyticsService() *AnalyticsService {
	return &AnalyticsService{}
}

func (s *AnalyticsService) GetData() *repository.AnalyticsData {
	return repository.GetAnalyticsData()
}
