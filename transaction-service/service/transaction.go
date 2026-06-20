package service

import "transaction-service/repository"

type TransactionService struct{}

func NewTransactionService() *TransactionService {
	return &TransactionService{}
}

func (s *TransactionService) GetTransactions() []repository.Transaction {
	return repository.GetTransactions()
}
