package repository

type Report struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Date   string `json:"date"`
	Author string `json:"author"`
	Status string `json:"status"`
}

func GetReports() []Report {
	return []Report{
		{ID: "#001", Title: "Q1 Revenue Summary", Date: "2024-03-31", Author: "alice@example.com", Status: "published"},
		{ID: "#002", Title: "User Acquisition — Feb 2024", Date: "2024-02-29", Author: "bob@example.com", Status: "published"},
		{ID: "#003", Title: "Infrastructure Cost Analysis", Date: "2024-04-15", Author: "carol@example.com", Status: "draft"},
		{ID: "#004", Title: "Churn Risk Cohort Study", Date: "2024-05-01", Author: "alice@example.com", Status: "draft"},
	}
}
