package database

type Rating struct {
    Day        string  `json:"day"`
    Category   string  `json:"category"`
    Score      int32   `json:"daily_score"`
    Total      int32   `json:"total"`
}