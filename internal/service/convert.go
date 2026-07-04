package service

import (
	pdpb "github.com/pspd-2026-2-trabalho-2/data-transform-service/gen/patientdata/v1"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/anonymize"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/fhir"
)

// toRaw converte o Patient do patient-data para a entrada do anonimizador.
func toRaw(p *pdpb.Patient) anonymize.RawPatient {
	return anonymize.RawPatient{
		ID:        p.GetPatientId(),
		FullName:  p.GetFullName(),
		BirthDate: p.GetBirthDate(),
		Gender:    p.GetGender(),
		City:      p.GetCity(),
		State:     p.GetState(),
		CPF:       p.GetCpf(),
		CNS:       p.GetCns(),
	}
}

func subject(id string) *fhir.Reference { return &fhir.Reference{Reference: "Patient/" + id} }

// patientResource mapeia a visão anonimizada para um recurso FHIR Patient.
func patientResource(v anonymize.PatientView) fhir.Patient {
	p := fhir.Patient{ResourceType: "Patient", ID: v.ID, Gender: v.Gender, BirthDate: v.BirthDate}
	if v.Name != "" {
		p.Name = []fhir.HumanName{{Text: v.Name}}
	}
	if v.CPF != "" {
		p.Identifier = append(p.Identifier, fhir.Identifier{System: "urn:oid:cpf", Value: v.CPF})
	}
	if v.CNS != "" {
		p.Identifier = append(p.Identifier, fhir.Identifier{System: "urn:oid:cns", Value: v.CNS})
	}
	if v.City != "" || v.State != "" {
		p.Address = []fhir.Address{{City: v.City, State: v.State}}
	}
	if v.AgeRange != "" {
		p.Extension = []fhir.Extension{{
			URL:         "http://pspd.unb.br/fhir/StructureDefinition/age-range",
			ValueString: v.AgeRange,
		}}
	}
	return p
}

func encounterResource(subjectID string, e *pdpb.Encounter) fhir.Encounter {
	enc := fhir.Encounter{
		ResourceType: "Encounter",
		ID:           e.GetEncounterId(),
		Status:       "finished",
		Subject:      subject(subjectID),
	}
	if e.GetEncounterType() != "" {
		enc.Type = []fhir.CodeableConcept{{Text: e.GetEncounterType()}}
	}
	if e.GetDepartment() != "" {
		enc.ServiceType = &fhir.CodeableConcept{Text: e.GetDepartment()}
	}
	if e.GetStartDate() != "" || e.GetEndDate() != "" {
		enc.Period = &fhir.Period{Start: e.GetStartDate(), End: e.GetEndDate()}
	}
	return enc
}

func conditionResource(subjectID string, e *pdpb.ClinicalEvent) fhir.Condition {
	return fhir.Condition{
		ResourceType: "Condition",
		ID:           e.GetEventId(),
		Code:         &fhir.CodeableConcept{Text: textOr(e.GetDescription(), e.GetCode())},
		Subject:      subject(subjectID),
		RecordedDate: e.GetEventDate(),
	}
}

func observationResource(subjectID string, e *pdpb.ClinicalEvent) fhir.Observation {
	o := fhir.Observation{
		ResourceType:      "Observation",
		ID:                e.GetEventId(),
		Status:            "final",
		Code:              &fhir.CodeableConcept{Text: textOr(e.GetDescription(), e.GetCode())},
		Subject:           subject(subjectID),
		EffectiveDateTime: e.GetEventDate(),
	}
	if e.Value != nil {
		o.ValueQuantity = &fhir.Quantity{Value: e.GetValue(), Unit: e.GetUnit()}
	}
	return o
}

func medicationResource(subjectID string, e *pdpb.ClinicalEvent) fhir.MedicationRequest {
	m := fhir.MedicationRequest{
		ResourceType:              "MedicationRequest",
		ID:                        e.GetEventId(),
		Status:                    "active",
		Intent:                    "order",
		MedicationCodeableConcept: &fhir.CodeableConcept{Text: textOr(e.GetDescription(), e.GetCode())},
		Subject:                   subject(subjectID),
		AuthoredOn:                e.GetEventDate(),
	}
	if e.Value != nil {
		m.DosageInstruction = []fhir.Dosage{{
			Text:        e.GetDescription(),
			DoseAndRate: []fhir.DoseAndRate{{DoseQuantity: &fhir.Quantity{Value: e.GetValue(), Unit: e.GetUnit()}}},
		}}
	}
	return m
}

// eventResource despacha um evento clínico para o recurso FHIR correspondente.
func eventResource(subjectID string, e *pdpb.ClinicalEvent) any {
	switch e.GetEventType() {
	case "Observation":
		return observationResource(subjectID, e)
	case "Medication":
		return medicationResource(subjectID, e)
	default: // Condition
		return conditionResource(subjectID, e)
	}
}

func researchStudyResource(p *pdpb.Project) fhir.ResearchStudy {
	return fhir.ResearchStudy{
		ResourceType: "ResearchStudy",
		ID:           p.GetProjectId(),
		Title:        p.GetTitle(),
		Status:       studyStatus(p.GetStatus()),
	}
}

func studyStatus(s string) string {
	switch s {
	case "Aprovado":
		return "active"
	case "Expirado":
		return "completed"
	case "Suspenso":
		return "suspended"
	default:
		return "active"
	}
}

func textOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
