package service

import (
	"context"
	"log"
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

const (
    MIN_MONTH_LENGTH = 28
    DATE_FORMAT      = "2006-01-02T15:04:05"
    SPELLING         = "Spelling"
    GRAMMAR          = "Grammar"
    GDPR             = "GDPR"
    RANDOMNESS       = "Randomness"
)

func NewRatingsService(repo *database.Repository) *RatingsService {
	return &RatingsService{repo: repo}
}

func (s *RatingsService) GetOverallScore(ctx context.Context, req *pb.OverallScoreRequest) (*pb.OverallScoreResponse, error) {
	startTime := req.StartDate.AsTime()
	endTime := req.EndDate.AsTime()
	log.Printf("Processing request: %v to %v", startTime, endTime)

	if req.StartDate == nil || req.EndDate == nil || startTime.After(endTime) {
		log.Printf("Invalid date range: %v to %v", startTime, endTime)
		return nil, status.Errorf(codes.InvalidArgument, "start_date and end_date are required, and start_date cannot be after end_date")
	}

	overallScore, err := s.repo.GetOverallScore(startTime.Format(DATE_FORMAT), endTime.Format(DATE_FORMAT))
	if err != nil {
		log.Printf("Failed to get overall score: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to retrieve overall score")
	}

	return &pb.OverallScoreResponse{
		OverallScore: overallScore,
	}, nil
}

func (s *RatingsService) GetAggregatedScores(ctx context.Context, req *pb.AggregatedScoresRequest) (*pb.AggregatedScoresResponse, error) {
	startTime := req.StartDate.AsTime()
	endTime := req.EndDate.AsTime()
	log.Printf("Processing request: %v to %v", startTime, endTime)

	if req.StartDate == nil || req.EndDate == nil || startTime.After(endTime) {
		log.Printf("Invalid date range: %v to %v", startTime, endTime)
		return nil, status.Errorf(codes.InvalidArgument, "start_date and end_date are required, and start_date cannot be after end_date")
	}

	ratings, err := s.repo.GetWeightedRatings(startTime.Format(DATE_FORMAT), endTime.Format(DATE_FORMAT))
	if err != nil {
		log.Printf("Failed to get ratings: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to retrieve ratings")
	}

	var report []*pb.Score

	ratingsScore, err := CalculateRatingsReport(ratings)
	if err != nil {
		log.Printf("Failed to calculate ratings report: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to calculate ratings report")
	}

	report = append(report, ratingsScore)

	if withinMinMonth(startTime, endTime) || withinCalendarMonth(startTime, endTime) {
		log.Printf("Generating daily report: %v to %v", startTime, endTime)
		dailyReport, err := CalculateDailyReport(ratings)

		if err != nil {
			log.Printf("Failed to calculate daily report: %v", err)
			return nil, status.Errorf(codes.Internal, "Failed to calculate daily report")
		}
		report = append(report, dailyReport...)
	} else {
		log.Printf("Generating weekly report: %v to %v", startTime, endTime)
		weeklyReport, err := CalculateWeeklyReport(ratings)
		if err != nil {
			log.Printf("Failed to calculate weekly report: %v", err)
			return nil, status.Errorf(codes.Internal, "Failed to calculate weekly report")
		}
		report = append(report, weeklyReport...)
	}

	return &pb.AggregatedScoresResponse{
		Scores: report,
	}, nil
}

func CalculateDailyReport(ratings []database.Rating) ([]*pb.Score, error) {
	if len(ratings) == 0 {
		return []*pb.Score{}, nil
	}

	var report []*pb.Score
	container := createEmptyContainer[int32]()

	score := &pb.Score{
		Type:  pb.ScoreEnum_DAILY,
		Value: ratings[0].Day,
	}

	for _, rating := range ratings {
		var err error
		if rating.Day == score.Value {
			container, err = scoreByCategory[int32](container, rating, func(v int32, r database.Rating) int32 { return r.Score })
			if err != nil {
				return nil, fmt.Errorf("failed to score rating: %w", err)
			}
		} else {
			report = includeRatingToReport(report, score, container)
			score.Value = rating.Day
			container = createEmptyContainer[int32]()

			container, err = scoreByCategory[int32](container, rating, func(v int32, r database.Rating) int32 { return r.Score })
			if err != nil {
				return nil, fmt.Errorf("failed to score rating: %w", err)
			}
		}
	}

	report = includeRatingToReport(report, score, container)

	return report, nil
}

func CalculateWeeklyReport(ratings []database.Rating) ([]*pb.Score, error) {
	if len(ratings) == 0 {
		return []*pb.Score{}, nil
	}
	
	var report []*pb.Score
	currentDay := ratings[0].Day
	dayCounter, weekNumber := int32(1), int32(1)
	container := createEmptyContainer[[]int32]()

	for _, rating := range ratings {
		var err error
		container, err = scoreByCategory[[]int32](container, rating, func(v []int32, r database.Rating) []int32 { return append(v, r.Score) })
		if err != nil {
			return nil, fmt.Errorf( "failed to score rating: %w", err)
		}

		if rating.Day != currentDay {
			dayCounter++
		}

		if dayCounter > 7 {
			dayCounter = 1
			weekNumber++
			report = append(report, aggregateWeeklyScores(container, weekNumber-1))
			container = createEmptyContainer[[]int32]()
		}

		currentDay = rating.Day
	}

	if dayCounter > 1 {
		report = append(report, aggregateWeeklyScores(container, weekNumber))
	}

	return report, nil
}

func CalculateRatingsReport(ratings []database.Rating) (*pb.Score, error) {
	container := createEmptyContainer[int32]()

	for _, rating := range ratings {
		var err error
		container, err = scoreByCategory[int32](container, rating, func(v int32, r database.Rating) int32 { return v + r.Total })
		if err != nil {
			return nil, fmt.Errorf("failed to score rating: %w", err)
		}
	}
	return &pb.Score{
		Type: pb.ScoreEnum_RATINGS,
		Spelling: container.Spelling,
		Grammar: container.Grammar,
		Gdpr: container.Gdpr,
		Randomness: container.Randomness,
	}, nil
}

func scoreByCategory[T ScoreContainerValue](container ScoreContainer[T], rating database.Rating, updateFunc func(T, database.Rating) T) (ScoreContainer[T], error) {
	switch rating.Category {
	case SPELLING:
		container.Spelling = updateFunc(container.Spelling, rating)
	case GRAMMAR:
		container.Grammar = updateFunc(container.Grammar, rating)
	case GDPR:
		container.Gdpr = updateFunc(container.Gdpr, rating)
	case RANDOMNESS:
		container.Randomness = updateFunc(container.Randomness, rating)
	default:
		log.Printf("unknown category: %s", rating.Category)
		return container, fmt.Errorf("unknown category: %s", rating.Category)
	}
	return container, nil
}

func aggregateWeeklyScores(container ScoreContainer[[]int32], weekNumber int32) *pb.Score {
	return &pb.Score{
		Type: pb.ScoreEnum_WEEKLY,
		Value: fmt.Sprintf("Week %d", weekNumber),
		Spelling: calculateAverage(container.Spelling),
		Grammar: calculateAverage(container.Grammar),
		Gdpr: calculateAverage(container.Gdpr),
		Randomness: calculateAverage(container.Randomness),
	}
}

func includeRatingToReport(report []*pb.Score, score *pb.Score, container ScoreContainer[int32]) []*pb.Score {
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
