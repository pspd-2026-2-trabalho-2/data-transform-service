// Package service aplica o nível de acesso sobre os dados recebidos e converte para HL7/FHIR.
package service

import (
	"math"

	pdpb "github.com/pspd-2026-2-trabalho-2/data-transform-service/gen/patientdata/v1"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/anonymize"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/fhir"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/observability"
)

// Service reúne o anonimizador e as métricas.
type Service struct {
	anon    *anonymize.Anonymizer
	metrics *observability.Metrics
}

// New cria o serviço.
func New(anon *anonymize.Anonymizer, m *observability.Metrics) *Service {
	return &Service{anon: anon, metrics: m}
}

// Percentage é uma distribuição com contagem e percentual.
type Percentage struct {
	Key        string
	Count      int64
	Percentage float64
}

// CohortStats é o resultado agregado (não-FHIR) de uma coorte.
type CohortStats struct {
	ConditionCode       string
	TotalPatients       int64
	BySex               []Percentage
	ByAgeRange          []Percentage
	MeanHbA1c           float64
	MedianHbA1c         float64
	MedicationFrequency []Percentage
}

// PatientExams associa um paciente aos seus exames (entrada de TransformCohortExams).
type PatientExams struct {
	Patient *pdpb.Patient
	Exams   []*pdpb.ClinicalEvent
}

// TransformPatient devolve um recurso FHIR Patient com o nível aplicado.
func (s *Service) TransformPatient(patient *pdpb.Patient, level anonymize.Level) (string, error) {
	s.metrics.RecordTransform("TransformPatient", level.String())
	return fhir.ToJSON(patientResource(s.anon.Patient(toRaw(patient), level)))
}

// TransformClinicalSummary devolve um Bundle com o resumo clínico.
func (s *Service) TransformClinicalSummary(sum *pdpb.ClinicalSummary, level anonymize.Level) (string, error) {
	view := s.anon.Patient(toRaw(sum.GetPatient()), level)
	resources := []any{patientResource(view)}
	if sum.GetLastEncounter() != nil {
		resources = append(resources, encounterResource(view.ID, sum.GetLastEncounter()))
	}
	for _, c := range sum.GetConditions() {
		resources = append(resources, conditionResource(view.ID, c))
	}
	for _, o := range sum.GetRecentObservations() {
		resources = append(resources, observationResource(view.ID, o))
	}
	for _, m := range sum.GetActiveMedications() {
		resources = append(resources, medicationResource(view.ID, m))
	}
	s.metrics.RecordTransform("TransformClinicalSummary", level.String())
	return fhir.ToJSON(fhir.NewBundle(resources...))
}

// TransformClinicalHistory devolve um Bundle com o histórico clínico temporal.
func (s *Service) TransformClinicalHistory(patient *pdpb.Patient, events []*pdpb.ClinicalEvent, level anonymize.Level) (string, error) {
	view := s.anon.Patient(toRaw(patient), level)
	resources := []any{patientResource(view)}
	for _, e := range events {
		resources = append(resources, eventResource(view.ID, e))
	}
	s.metrics.RecordTransform("TransformClinicalHistory", level.String())
	return fhir.ToJSON(fhir.NewBundle(resources...))
}

// TransformPatientList devolve um Bundle de Patient (lista de um médico/estagiário).
func (s *Service) TransformPatientList(patients []*pdpb.Patient, level anonymize.Level) (string, error) {
	resources := make([]any, 0, len(patients))
	for _, p := range patients {
		resources = append(resources, patientResource(s.anon.Patient(toRaw(p), level)))
	}
	s.metrics.RecordTransform("TransformPatientList", level.String())
	return fhir.ToJSON(fhir.NewBundle(resources...))
}

// TransformCohortExams devolve um Bundle com pacientes da coorte e seus exames,
// tipicamente com nível ANONYMIZED (para pesquisadores).
func (s *Service) TransformCohortExams(items []PatientExams, level anonymize.Level) (string, error) {
	resources := make([]any, 0, len(items)*2)
	for _, item := range items {
		view := s.anon.Patient(toRaw(item.Patient), level)
		resources = append(resources, patientResource(view))
		for _, o := range item.Exams {
			resources = append(resources, observationResource(view.ID, o))
		}
	}
	s.metrics.RecordTransform("TransformCohortExams", level.String())
	return fhir.ToJSON(fhir.NewBundle(resources...))
}

// TransformProjects devolve um Bundle de ResearchStudy.
func (s *Service) TransformProjects(projects []*pdpb.Project) (string, error) {
	resources := make([]any, 0, len(projects))
	for _, p := range projects {
		resources = append(resources, researchStudyResource(p))
	}
	s.metrics.RecordTransform("TransformProjects", "FULL")
	return fhir.ToJSON(fhir.NewBundle(resources...))
}

// TransformCohortStatistics converte as contagens cruas em percentuais (AGGREGATED).
func (s *Service) TransformCohortStatistics(stats *pdpb.CohortStatistics) *CohortStats {
	total := stats.GetTotalPatients()
	s.metrics.RecordTransform("TransformCohortStatistics", anonymize.LevelAggregated.String())
	return &CohortStats{
		ConditionCode:       stats.GetConditionCode(),
		TotalPatients:       total,
		BySex:               toPercentages(stats.GetBySex(), total),
		ByAgeRange:          toPercentages(stats.GetByAgeRange(), total),
		MeanHbA1c:           stats.GetMeanHba1C(),
		MedianHbA1c:         stats.GetMedianHba1C(),
		MedicationFrequency: toPercentages(stats.GetMedicationFrequency(), total),
	}
}

func toPercentages(counts []*pdpb.Count, total int64) []Percentage {
	out := make([]Percentage, 0, len(counts))
	for _, c := range counts {
		pct := 0.0
		if total > 0 {
			pct = math.Round(float64(c.GetCount())/float64(total)*10000) / 100
		}
		out = append(out, Percentage{Key: c.GetKey(), Count: c.GetCount(), Percentage: pct})
	}
	return out
}
