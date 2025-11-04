package service

import (
	"time"
)

type ScoreContainerValue interface {
	int32 | []int32
}

type ScoreContainer[T ScoreContainerValue] struct {
	Spelling   T
	Grammar    T
	Gdpr       T
	Randomness T
}

func createEmptyContainer[T ScoreContainerValue]() ScoreContainer[T] {
	var zero T
	return ScoreContainer[T]{
		Spelling:   zero,
		Grammar:    zero,
		Gdpr:       zero,
		Randomness: zero,
	}
}

func withinCalendarMonth(start, end time.Time) bool {
	return start.Year() == end.Year() && start.Month() == end.Month()
}

func withinMinMonth(start, end time.Time) bool {
	thirtyOneDaysAgo := end.AddDate(0, 0, -MIN_MONTH_LENGTH)
	return start.After(thirtyOneDaysAgo)
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
