package service

import (
	"context"
	"testing"
	"time"
	"helpdesk-ratings/internal/database"
	pb "helpdesk-ratings/proto/gen"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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