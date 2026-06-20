package service

import "transaction/repository"

type TransactionService struct{}

func NewTransactionService() *TransactionService {
	return &TransactionService{}
}

func (s *TransactionService) GetTransactions() []repository.Transaction {
	return repository.GetTransactions()
}
