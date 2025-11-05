package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

type Repository struct {
	db *sql.DB
}

type Rating struct {
    Day        string  `json:"day"`
    Category   string  `json:"category"`
		Value      int32   `json:"value"`
    Weight     float64 `json:"weight"`
}

func NewRepository(dataSourceName string) (*Repository, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) GetOverallScore(startDate, endDate string) (float32, error) {
	query := `
		SELECT COALESCE(100.0 * SUM((r.rating / 5.0) * rc.weight) / SUM(rc.weight), 0) AS overall_score
        FROM ratings r
        JOIN rating_categories rc ON rc.id = r.rating_category_id
        WHERE r.created_at BETWEEN ? AND ?`
	var overallScore float32
	err := r.db.QueryRow(query, startDate, endDate).Scan(&overallScore)
	if err != nil {
		return 0, err
	}
	return overallScore, nil
}

func (r *Repository) GetWeightedRatings(startDate, endDate string) ([]Rating, error) {
	query := `
		SELECT DATE(r.created_at) AS day, rc.name as category, r.rating as value, rc.weight as weight
		FROM ratings r
		JOIN rating_categories rc ON rc.id = r.rating_category_id
			WHERE r.created_at BETWEEN ? AND ?
			ORDER BY day, r.rating_category_id`

	rows, err := r.db.Query(query, startDate, endDate)
	if err != nil {
			return nil, err
	}
	defer rows.Close()

	var ratings []Rating
	for rows.Next() {
		var day, category string
		var weight float64
		var value int32

		err := rows.Scan(&day, &category, &value, &weight)
		if err != nil {
			return nil, err
		}

		ratings = append(ratings, Rating{
			Day:      day,
			Category: category,
			Value:    value,
			Weight:   weight,
		})
	}
	
	return ratings, rows.Err()
}
