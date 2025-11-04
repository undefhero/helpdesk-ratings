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

type ScoreContainerValue interface {
	int32 | []int32
}

type ScoreContainer[T ScoreContainerValue] struct {
	Spelling   T
	Grammar    T
	Gdpr       T
	Randomness T
}

const MIN_MONTH_LENGTH = 28

func NewRatingsService(repo *database.Repository) *RatingsService {
	return &RatingsService{repo: repo}
}

func (s *RatingsService) GetOverallScore(ctx context.Context, req *pb.OverallScoreRequest) (*pb.OverallScoreResponse, error) {
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

	overallScore, err := s.repo.GetOverallScore(startTime.Format("2006-01-02T15:04:05"), endTime.Format("2006-01-02T15:04:05"))
	if err != nil {
		log.Fatalf("Failed to get overall score: %v", err)
	}

	return &pb.OverallScoreResponse{
		OverallScore: overallScore,
	}, nil
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
	var container, _ = createEmptyContainer[int32]()

	for _, rating := range ratings {
		if score == nil {
			score = &pb.Score{
				Type:  pb.ScoreEnum_DAILY,
				Value: rating.Day,
			}

			container, _ = scoreByCategory(container, rating, func(s int32) int32 { return s })
		} else if score != nil && rating.Day == score.Value {
			container, _ = scoreByCategory(container, rating, func(s int32) int32 { return s })
		} else if score != nil && rating.Day != score.Value {
			report = includeRatingReport(report, score, container)
			score.Value = rating.Day
			container, _ = createEmptyContainer[int32]()
			container, _ = scoreByCategory(container, rating, func(s int32) int32 { return s })
		}
	}

	if score != nil {
		report = includeRatingReport(report, score, container)
	}

	return report, nil
}

func CalculateRatingsReport(ratings []database.Rating) (*pb.Score, error) {
	var container, _ = createEmptyContainer[int32]()
	for _, rating := range ratings {
		container, _ = scoreByCategory(container, rating, func(s int32) int32 { return s + rating.Score })
	}
	return &pb.Score{
		Type:  			pb.ScoreEnum_RATINGS,
		Spelling:   container.Spelling,
		Grammar:    container.Grammar,
		Gdpr:       container.Gdpr,
		Randomness: container.Randomness,
	}, nil
}

func scoreByCategory[T ScoreContainerValue](container ScoreContainer[T], rating database.Rating, updateFunc func(int32) T) (ScoreContainer[T], error) {
	switch rating.Category {
	case "Spelling":
		container.Spelling = updateFunc(rating.Score)
	case "Grammar":
		container.Grammar = updateFunc(rating.Score)
	case "GDPR":
		container.Gdpr = updateFunc(rating.Score)
	case "Randomness":
		container.Randomness = updateFunc(rating.Score)
	default:
		log.Printf("unknown category: %s", rating.Category)
		return container, fmt.Errorf("unknown category: %s", rating.Category)
	}
	return container, nil
}

func createEmptyContainer[T ScoreContainerValue]() (ScoreContainer[T], error) {
	var zero T
	return ScoreContainer[T]{
		Spelling:   zero,
		Grammar:    zero,
		Gdpr:       zero,
		Randomness: zero,
	}, nil
}

func withinCalendarMonth(start, end time.Time) bool {
	return start.Year() == end.Year() && start.Month() == end.Month()
}

func withinMinMonth(start, end time.Time) bool {
	thirtyOneDaysAgo := end.AddDate(0, 0, -MIN_MONTH_LENGTH)
	return start.After(thirtyOneDaysAgo)
}

func includeRatingReport(report []*pb.Score, score *pb.Score, container ScoreContainer[int32]) []*pb.Score {
	if score == nil {
		return report
	}

	score.Spelling = container.Spelling
	score.Grammar = container.Grammar
	score.Gdpr = container.Gdpr
	score.Randomness = container.Randomness
	report = append(report, score)

	return report
}