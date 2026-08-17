// Package grpcapi реализует gRPC ScannerService поверх application/scanner.
package grpcapi

import (
	"bytes"
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"configguard/internal/application/scanner"
	"configguard/internal/domain/issue"
	pb "configguard/internal/genproto/configguard/v1"
	"configguard/internal/infrastructure/parser"
)

type handler struct {
	pb.UnimplementedScannerServiceServer
	svc *scanner.Service
}

func (h *handler) Scan(_ context.Context, req *pb.ScanRequest) (*pb.ScanResponse, error) {
	format := formatFromProto(req.GetFormat())
	report, err := h.svc.ScanReader(bytes.NewReader(req.GetConfig()), "request", format)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &pb.ScanResponse{
		Issues:    toProtoIssues(report.Issues),
		HasIssues: report.HasIssues(),
	}, nil
}

func formatFromProto(f pb.Format) parser.Format {
	switch f {
	case pb.Format_FORMAT_JSON:
		return parser.JSON
	case pb.Format_FORMAT_YAML:
		return parser.YAML
	default:
		return parser.Unknown
	}
}

func toProtoIssues(issues []issue.Issue) []*pb.Issue {
	out := make([]*pb.Issue, 0, len(issues))
	for _, i := range issues {
		out = append(out, &pb.Issue{
			RuleId:         i.RuleID,
			Level:          levelToProto(i.Level),
			Field:          i.Field,
			Message:        i.Message,
			Recommendation: i.Recommendation,
		})
	}
	return out
}

func levelToProto(l issue.Level) pb.Level {
	switch l {
	case issue.Low:
		return pb.Level_LEVEL_LOW
	case issue.Medium:
		return pb.Level_LEVEL_MEDIUM
	case issue.High:
		return pb.Level_LEVEL_HIGH
	default:
		return pb.Level_LEVEL_UNSPECIFIED
	}
}
