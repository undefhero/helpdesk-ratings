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
	} else {
		log.Printf("Generating weekly report: %v to %v", startTime, endTime)
		
		weeklyReport, err := CalculateWeeklyReport(ratings)
		if err != nil {
			log.Fatalf("Failed to calculate weekly report: %v", err)
		}
		report = append(report, weeklyReport...)
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

			container, _ = scoreByCategory[int32](container, rating, func(v int32, r database.Rating) int32 { return r.Score })
		} else if score != nil && rating.Day == score.Value {
			container, _ = scoreByCategory[int32](container, rating, func(v int32, r database.Rating) int32 { return r.Score })
		} else if score != nil && rating.Day != score.Value {
			report = includeRatingToReport(report, score, container)
			score.Value = rating.Day
			container, _ = createEmptyContainer[int32]()
			container, _ = scoreByCategory[int32](container, rating, func(v int32, r database.Rating) int32 { return r.Score })
		}
	}

	if score != nil {
		report = includeRatingToReport(report, score, container)
	}

	return report, nil
}

func CalculateWeeklyReport(ratings []database.Rating) ([]*pb.Score, error) {
	var report []*pb.Score
	var rating database.Rating
	var currentDay string = ratings[0].Day
	var dayCounter, weekNumber int32 = 1, 1
	var container, _ = createEmptyContainer[[]int32]()
	
	for dayCounter <= 7 && len(ratings) > 0 {
		rating = ratings[0]
		ratings = ratings[1:]
		container, _ = scoreByCategory[[]int32](container, rating, func(v []int32, r database.Rating) []int32 { return append(v, r.Score) })

		if rating.Day != currentDay {
			dayCounter++
		}

		if dayCounter > 7 {
			dayCounter = 1
			weekNumber++
			report = append(report, aggreagateWeeklyScores(container, weekNumber-1))
			container, _ = createEmptyContainer[[]int32]()
		}

		currentDay = rating.Day
	}

	if dayCounter > 1 {
		report = append(report, aggreagateWeeklyScores(container, weekNumber))
	}

	return report, nil
}

func CalculateRatingsReport(ratings []database.Rating) (*pb.Score, error) {
	var container, _ = createEmptyContainer[int32]()
	for _, rating := range ratings {
		container, _ = scoreByCategory[int32](container, rating, func(v int32, r database.Rating) int32 { return v + r.Total })
	}
	return &pb.Score{
		Type:  			pb.ScoreEnum_RATINGS,
		Spelling:   container.Spelling,
		Grammar:    container.Grammar,
		Gdpr:       container.Gdpr,
		Randomness: container.Randomness,
	}, nil
}

func scoreByCategory[T ScoreContainerValue](container ScoreContainer[T], rating database.Rating, updateFunc func(T, database.Rating) T) (ScoreContainer[T], error) {
	switch rating.Category {
	case "Spelling":
		container.Spelling = updateFunc(container.Spelling, rating)
	case "Grammar":
		container.Grammar = updateFunc(container.Grammar, rating)
	case "GDPR":
		container.Gdpr = updateFunc(container.Gdpr, rating)
	case "Randomness":
		container.Randomness = updateFunc(container.Randomness, rating)
	default:
		log.Printf("unknown category: %s", rating.Category)
		return container, fmt.Errorf("unknown category: %s", rating.Category)
	}
	return container, nil
}

func withinCalendarMonth(start, end time.Time) bool {
	return start.Year() == end.Year() && start.Month() == end.Month()
}

func withinMinMonth(start, end time.Time) bool {
	thirtyOneDaysAgo := end.AddDate(0, 0, -MIN_MONTH_LENGTH)
	return start.After(thirtyOneDaysAgo)
}

func aggreagateWeeklyScores(container ScoreContainer[[]int32], weekNumber int32) *pb.Score {
	return &pb.Score{
		Type:  			pb.ScoreEnum_WEEKLY,
		Value: 			fmt.Sprintf("Week %d", weekNumber),
		Spelling:   calculateAverage(container.Spelling),
		Grammar:    calculateAverage(container.Grammar),
		Gdpr:       calculateAverage(container.Gdpr),
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

func createEmptyContainer[T ScoreContainerValue]() (ScoreContainer[T], error) {
	var zero T
	return ScoreContainer[T]{
		Spelling:   zero,
		Grammar:    zero,
		Gdpr:       zero,
		Randomness: zero,
	}, nil
}

func calculateAverage(nums []int32) int32 {
	if len(nums) == 0 {
		return 0
	}
	var sum int32
	for _, n := range nums {
		sum += n
	}
	return sum / int32(len(nums))
}
