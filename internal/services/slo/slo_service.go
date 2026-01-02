package slo

import (
	"fmt"
	"time"
)

const (
	AvailabilitySLO = 0.999 // 99.9%
)

var latencySLOs = map[string]time.Duration{
	"/ondc/search":  500 * time.Millisecond,
	"/ondc/confirm": 1000 * time.Millisecond,
	"/ondc/status":  200 * time.Millisecond,
	"/ondc/track":   200 * time.Millisecond,
	"callback":      2000 * time.Millisecond,
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetLatencySLO(endpoint string) time.Duration {
	if slo, exists := latencySLOs[endpoint]; exists {
		return slo
	}
	return 0
}

func (s *Service) CheckLatencySLO(endpoint string, p95Latency time.Duration) (pass bool, reason string) {
	slo := s.GetLatencySLO(endpoint)
	if slo == 0 {
		return true, ""
	}

	if p95Latency > slo {
		return false, fmt.Sprintf("p95 latency %v exceeds SLO %v", p95Latency, slo)
	}

	return true, ""
}

func (s *Service) GetAvailabilitySLO() float64 {
	return AvailabilitySLO
}

func (s *Service) CheckAvailabilitySLO(availability float64) (pass bool, reason string) {
	if availability < AvailabilitySLO {
		return false, fmt.Sprintf("availability %.1f%% below SLO %.1f%%", availability*100, AvailabilitySLO*100)
	}
	return true, ""
}
