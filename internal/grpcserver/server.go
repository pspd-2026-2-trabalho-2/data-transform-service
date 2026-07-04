// Package grpcserver implementa o transporte gRPC do data-transform-service.
package grpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dtpb "github.com/pspd-2026-2-trabalho-2/data-transform-service/gen/datatransform/v1"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/anonymize"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/service"
)

// Server implementa dtpb.DataTransformServiceServer.
type Server struct {
	dtpb.UnimplementedDataTransformServiceServer
	svc *service.Service
}

// New cria o servidor gRPC.
func New(svc *service.Service) *Server { return &Server{svc: svc} }

func (s *Server) TransformPatient(_ context.Context, req *dtpb.TransformPatientRequest) (*dtpb.FhirResponse, error) {
	if req.GetPatient() == nil {
		return nil, status.Error(codes.InvalidArgument, "patient é obrigatório")
	}
	j, err := s.svc.TransformPatient(req.GetPatient(), toLevel(req.GetAccessLevel()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &dtpb.FhirResponse{FhirJson: j}, nil
}

func (s *Server) TransformClinicalSummary(_ context.Context, req *dtpb.TransformClinicalSummaryRequest) (*dtpb.FhirResponse, error) {
	if req.GetSummary() == nil || req.GetSummary().GetPatient() == nil {
		return nil, status.Error(codes.InvalidArgument, "summary com patient é obrigatório")
	}
	j, err := s.svc.TransformClinicalSummary(req.GetSummary(), toLevel(req.GetAccessLevel()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &dtpb.FhirResponse{FhirJson: j}, nil
}

func (s *Server) TransformClinicalHistory(_ context.Context, req *dtpb.TransformClinicalHistoryRequest) (*dtpb.FhirResponse, error) {
	if req.GetPatient() == nil {
		return nil, status.Error(codes.InvalidArgument, "patient é obrigatório")
	}
	j, err := s.svc.TransformClinicalHistory(req.GetPatient(), req.GetEvents(), toLevel(req.GetAccessLevel()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &dtpb.FhirResponse{FhirJson: j}, nil
}

func (s *Server) TransformPatientList(_ context.Context, req *dtpb.TransformPatientListRequest) (*dtpb.FhirResponse, error) {
	j, err := s.svc.TransformPatientList(req.GetPatients(), toLevel(req.GetAccessLevel()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &dtpb.FhirResponse{FhirJson: j}, nil
}

func (s *Server) TransformCohortExams(_ context.Context, req *dtpb.TransformCohortExamsRequest) (*dtpb.FhirResponse, error) {
	level := toLevel(req.GetAccessLevel())
	if level == anonymize.LevelFull { // coorte de pesquisa nunca é FULL
		level = anonymize.LevelAnonymized
	}
	items := make([]service.PatientExams, 0, len(req.GetPatients()))
	for _, pe := range req.GetPatients() {
		items = append(items, service.PatientExams{Patient: pe.GetPatient(), Exams: pe.GetExams()})
	}
	j, err := s.svc.TransformCohortExams(items, level)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &dtpb.FhirResponse{FhirJson: j}, nil
}

func (s *Server) TransformCohortStatistics(_ context.Context, req *dtpb.TransformCohortStatisticsRequest) (*dtpb.CohortStatisticsResponse, error) {
	if req.GetStats() == nil {
		return nil, status.Error(codes.InvalidArgument, "stats é obrigatório")
	}
	st := s.svc.TransformCohortStatistics(req.GetStats())
	return &dtpb.CohortStatisticsResponse{
		ConditionCode:       st.ConditionCode,
		TotalPatients:       st.TotalPatients,
		BySex:               toPBPercentages(st.BySex),
		ByAgeRange:          toPBPercentages(st.ByAgeRange),
		MeanHba1C:           st.MeanHbA1c,
		MedianHba1C:         st.MedianHbA1c,
		MedicationFrequency: toPBPercentages(st.MedicationFrequency),
	}, nil
}

func (s *Server) TransformProjects(_ context.Context, req *dtpb.TransformProjectsRequest) (*dtpb.FhirResponse, error) {
	j, err := s.svc.TransformProjects(req.GetProjects())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &dtpb.FhirResponse{FhirJson: j}, nil
}

func toLevel(l dtpb.AccessLevel) anonymize.Level {
	switch l {
	case dtpb.AccessLevel_PARTIAL:
		return anonymize.LevelPartial
	case dtpb.AccessLevel_ANONYMIZED:
		return anonymize.LevelAnonymized
	case dtpb.AccessLevel_AGGREGATED:
		return anonymize.LevelAggregated
	default:
		return anonymize.LevelFull
	}
}

func toPBPercentages(in []service.Percentage) []*dtpb.Percentage {
	out := make([]*dtpb.Percentage, 0, len(in))
	for _, p := range in {
		out = append(out, &dtpb.Percentage{Key: p.Key, Count: p.Count, Percentage: p.Percentage})
	}
	return out
}
