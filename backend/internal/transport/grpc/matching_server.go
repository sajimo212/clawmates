package grpc

import (
	"context"

	matchingalgo "clawmates/backend/internal/matching"

	matchingv1 "clawmates/backend/gen/matching/v1"
)

type MatchingServer struct {
	matchingv1.UnimplementedMatchingServiceServer
}

func NewMatchingServer() *MatchingServer {
	return &MatchingServer{}
}

func (s *MatchingServer) CalculateMatches(ctx context.Context, req *matchingv1.CalculateMatchesRequest) (*matchingv1.CalculateMatchesResponse, error) {
	matches := matchingalgo.CalculateMatches(req.Agents, req.ExistingPairs)
	return &matchingv1.CalculateMatchesResponse{Matches: matches}, nil
}
