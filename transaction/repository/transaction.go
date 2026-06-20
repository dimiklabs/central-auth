package repository

type Transaction struct {
	ID          string `json:"id"`
	Date        string `json:"date"`
	Description string `json:"description"`
	Amount      string `json:"amount"`
	Type        string `json:"type"`
	Status      string `json:"status"`
}

func GetTransactions() []Transaction {
	return []Transaction{
		{ID: "TXN-8821", Date: "2024-05-10", Description: "Stripe payout — April", Amount: "+$14,220.00", Type: "credit", Status: "settled"},
		{ID: "TXN-8820", Date: "2024-05-09", Description: "AWS — monthly bill", Amount: "-$3,412.55", Type: "debit", Status: "settled"},
		{ID: "TXN-8819", Date: "2024-05-08", Description: "Contractor payment — design", Amount: "-$2,500.00", Type: "debit", Status: "pending"},
		{ID: "TXN-8818", Date: "2024-05-07", Description: "Stripe payout — mid-month", Amount: "+$8,100.00", Type: "credit", Status: "settled"},
		{ID: "TXN-8817", Date: "2024-05-06", Description: "Refund — order #44291", Amount: "-$149.00", Type: "debit", Status: "failed"},
	}
}
