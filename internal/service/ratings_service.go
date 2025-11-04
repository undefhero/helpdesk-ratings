package service

import (
	"context"
	"log"
	"time"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"helpdesk-ratings/internal/database"
	pb "helpdesk-ratings/proto/gen"
)

type RatingsService struct {
	pb.UnimplementedServiceServer
	repo *database.Repository
}

const MIN_MONTH_LENGTH = 28

func NewRatingsService(repo *database.Repository) *RatingsService {
	return &RatingsService{repo: repo}
}

func (s *RatingsService) GetAggregatedScores(ctx context.Context, req *pb.AggregatedScoresRequest) (*pb.AggregatedScoresResponse, error) {
	startTime := req.StartDate.AsTime()
	endTime := req.EndDate.AsTime()
	log.Printf("Processing request: %v to %v", startTime, endTime)

	if req.StartDate == nil || req.EndDate == nil {
		log.Printf("Invalid date range: %v to %v", startTime, endTime)
		return nil, status.Errorf(codes.InvalidArgument, "start_date and end_date are required")
	}

	if startTime.After(endTime) {
		log.Printf("Invalid date range: %v to %v", startTime, endTime)
		return nil, status.Errorf(codes.InvalidArgument, "start_date cannot be after end_date")
	}

	ratings, err := s.repo.GetWeightedRatings(startTime.Format("2006-01-02T15:04:05"), endTime.Format("2006-01-02T15:04:05"))
	if err != nil {
		log.Fatalf("Failed to get ratings: %v", err)
	}

	var report []*pb.Score

	ratingsScore, err := CalculateRatingsReport(ratings)
	if err != nil {
		log.Fatalf("Failed to calculate ratings report: %v", err)
	}
	report = append(report, ratingsScore)

	if withinMinMonth(startTime, endTime) || withinCalendarMonth(startTime, endTime) {
		log.Printf("Generating daily report: %v to %v", startTime, endTime)

		dailyReport, err := CalculateDailyReport(ratings)
		if err != nil {
			log.Fatalf("Failed to calculate daily report: %v", err)
		}
		report = append(report, dailyReport...)
	}

	return &pb.AggregatedScoresResponse{
		Scores: report,
	}, nil
}


func CalculateDailyReport(ratings []database.Rating) ([]*pb.Score, error) {
	var report []*pb.Score
	var score *pb.Score

	for _, rating := range ratings {
		if score == nil {
			score = &pb.Score{
				Type:  pb.ScoreEnum_DAILY,
				Value: rating.Day,
			}

			scoreByCategory(score, rating, func(s int32) int32 { return s })
		} else if score != nil && rating.Day == score.Value {
			scoreByCategory(score, rating, func(s int32) int32 { return s })
		} else if score != nil && rating.Day != score.Value {
			report = append(report, score)
			score.Value = rating.Day
			scoreByCategory(score, rating, func(s int32) int32 { return s })
		}
	}

	if score != nil {
		report = append(report, score)
	}

	return report, nil
}

func CalculateRatingsReport(ratings []database.Rating) (*pb.Score, error) {
	var score = &pb.Score{
		Type:  pb.ScoreEnum_RATINGS,
		Spelling:   0,
		Grammar:    0,
		Gdpr:       0,
		Randomness: 0,
	}

	for _, rating := range ratings {
		scoreByCategory(score, rating, func (s int32) int32 { return s + rating.Score })
	}
	return score, nil
}

func scoreByCategory(score *pb.Score, rating database.Rating, updateFunc func(int32) int32) (*pb.Score, error) {
	switch rating.Category {
	case "Spelling":
		score.Spelling = updateFunc(rating.Score)
	case "Grammar":
		score.Grammar = updateFunc(rating.Score)
	case "GDPR":
		score.Gdpr = updateFunc(rating.Score)
	case "Randomness":
		score.Randomness = updateFunc(rating.Score)
	default:
		log.Printf("unknown category: %s", rating.Category)
		return nil, fmt.Errorf("unknown category: %s", rating.Category)
	}
	return score, nil
}

func withinCalendarMonth(start, end time.Time) bool {
	return start.Year() == end.Year() && start.Month() == end.Month()
}

func withinMinMonth(start, end time.Time) bool {
	thirtyOneDaysAgo := end.AddDate(0, 0, -MIN_MONTH_LENGTH)
	return start.After(thirtyOneDaysAgo)
}