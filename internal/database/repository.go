package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

type Repository struct {
	db *sql.DB
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

func (r *Repository) GetWeightedRatings(startDate, endDate string) ([]Rating, error) {
	query := `
		SELECT DATE(r.created_at) AS day, rc.name as category, 
			COALESCE(100.0 * SUM((r.rating / 5.0) * rc.weight) / SUM(rc.weight), 0) AS daily_score, COUNT(r.id) as total
		FROM ratings r
		JOIN rating_categories rc ON rc.id = r.rating_category_id
			WHERE r.created_at BETWEEN ? AND ?
			GROUP BY day, r.rating_category_id 
			ORDER BY day, r.rating_category_id`

	rows, err := r.db.Query(query, startDate, endDate)
	if err != nil {
			return nil, err
	}
	defer rows.Close()

	var ratings []Rating
	for rows.Next() {
		var day, category string
		var score float64
		var total int32

		err := rows.Scan(&day, &category, &score, &total)
		if err != nil {
			return nil, err
		}

		ratings = append(ratings, Rating{
			Day:      day,
			Category: category,
			Score:    int32(score),
			Total:    total,
		})
	}
	
	return ratings, rows.Err()
}
