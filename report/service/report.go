package service

import "report/repository"

type ReportService struct{}

func NewReportService() *ReportService {
	return &ReportService{}
}

func (s *ReportService) GetReports() []repository.Report {
	return repository.GetReports()
}
