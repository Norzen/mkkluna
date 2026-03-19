package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sony/gobreaker/v2"
)

type EmailService interface {
	SendInvitation(ctx context.Context, email string, teamID int64) error
}

type emailService struct {
	cb *gobreaker.CircuitBreaker[any]
}

func NewEmailService() EmailService {
	cbSettings := gobreaker.Settings{
		Name:        "email-service",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Printf("circuit breaker %s: %s -> %s", name, from, to)
		},
	}

	return &emailService{
		cb: gobreaker.NewCircuitBreaker[any](cbSettings),
	}
}

func (s *emailService) SendInvitation(ctx context.Context, email string, teamID int64) error {
	_, err := s.cb.Execute(func() (any, error) {
		return nil, s.doSendInvitation(ctx, email, teamID)
	})
	return err
}

func (s *emailService) doSendInvitation(_ context.Context, email string, teamID int64) error {
	log.Printf("[EMAIL MOCK] Sending invitation to %s for team %d", email, teamID)

	// Simulate external service call latency
	time.Sleep(50 * time.Millisecond)

	if email == "fail@test.com" {
		return fmt.Errorf("mock email service failure")
	}

	log.Printf("[EMAIL MOCK] Invitation sent to %s", email)
	return nil
}
