package service

import "analytics-service/repository"

type AnalyticsService struct{}

func NewAnalyticsService() *AnalyticsService {
	return &AnalyticsService{}
}

func (s *AnalyticsService) GetData() *repository.AnalyticsData {
	return repository.GetAnalyticsData()
}
