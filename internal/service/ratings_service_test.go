package service

import (
	"testing"
	"context"
	"time"
	"helpdesk-ratings/internal/database"
	pb "helpdesk-ratings/proto/gen"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetOverallScore(t *testing.T) {
	repo, err := database.NewRepository("../../database.db")
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()
	
	ratingsService := NewRatingsService(repo)
	req := &pb.OverallScoreRequest{
		StartDate: timestamppb.New(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:   timestamppb.New(time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC)),
	}

	response, err := ratingsService.GetOverallScore(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	t.Logf("Overall Score: %v", response.OverallScore)

	if response != nil && (response.OverallScore < 0 || response.OverallScore > 100) {
		t.Fatalf("Expected overall score between 0 and 100, got %v", response.OverallScore)
	}
}

func TestGetAggregatedScoresDaily(t *testing.T) {
	repo, err := database.NewRepository("../../database.db")
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ratingsService := NewRatingsService(repo)

	req := &pb.AggregatedScoresRequest{
		StartDate: timestamppb.New(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:   timestamppb.New(time.Date(2025, 1, 28, 23, 59, 59, 0, time.UTC)),
	}

	response, err := ratingsService.GetAggregatedScores(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response != nil && len(response.Scores) != 29 {
		t.Fatal("Expected 29 scores")
	}

	if response.Scores[0].Value == response.Scores[1].Value {
		t.Fatalf("Expected different scores for different days, got same score")
	}
}

func TestGetAggregatedScoresWeekly(t *testing.T) {
	repo, err := database.NewRepository("../../database.db")
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ratingsService := NewRatingsService(repo)

	req := &pb.AggregatedScoresRequest{
		StartDate: timestamppb.New(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:   timestamppb.New(time.Date(2025, 2, 1, 23, 59, 59, 0, time.UTC)),
	}

	response, err := ratingsService.GetAggregatedScores(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if len(response.Scores) == 1 {
		t.Fatal("Only ratings report returned")
	}
}

func TestCalculateWeightedScore(t *testing.T) {
	scores := []ScoreType{
    {Value: 4, Weight: 0.7},
    {Value: 5, Weight: 0.3},
	}

	expected := int32(86)
	result := calculateWeightedScore(scores)

	if result != expected {
		t.Fatalf("Expected %v, got %v", expected, result)
	}
}
