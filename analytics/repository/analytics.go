package repository

type Stat struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Delta string `json:"delta"`
	Up    bool   `json:"up"`
}

type Channel struct {
	Name     string `json:"name"`
	Sessions int    `json:"sessions"`
	Share    int    `json:"share"`
}

type AnalyticsData struct {
	Stats    []Stat    `json:"stats"`
	Channels []Channel `json:"channels"`
}

func GetAnalyticsData() *AnalyticsData {
	return &AnalyticsData{
		Stats: []Stat{
			{Label: "Monthly Active Users", Value: "24,812", Delta: "↑ 12% vs last month", Up: true},
			{Label: "Avg. Session Length", Value: "4m 32s", Delta: "↑ 8% vs last month", Up: true},
			{Label: "Conversion Rate", Value: "3.7%", Delta: "↑ 0.4 pp vs last month", Up: true},
			{Label: "Bounce Rate", Value: "41%", Delta: "↑ 2 pp vs last month", Up: false},
		},
		Channels: []Channel{
			{Name: "Organic Search", Sessions: 11240, Share: 45},
			{Name: "Direct", Sessions: 7445, Share: 30},
			{Name: "Referral", Sessions: 3722, Share: 15},
			{Name: "Paid Search", Sessions: 2405, Share: 10},
		},
	}
}
