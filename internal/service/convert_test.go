package service

import (
	"strings"
	"testing"

	pdpb "github.com/pspd-2026-2-trabalho-2/data-transform-service/gen/patientdata/v1"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/anonymize"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/fhir"
)

func rawJoao() anonymize.RawPatient {
	return anonymize.RawPatient{
		ID: "P000001", FullName: "João da Silva", BirthDate: "1970-05-10",
		Gender: "male", City: "Brasilia", State: "DF",
		CPF: "111.111.111-11", CNS: "700000000000001",
	}
}

func TestPatientResourceFullJSON(t *testing.T) {
	a := anonymize.New("salt")
	res := patientResource(a.Patient(rawJoao(), anonymize.LevelFull))
	j, err := fhir.ToJSON(res)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"resourceType": "Patient"`, `"João da Silva"`, `"birthDate": "1970-05-10"`, `"111.111.111-11"`} {
		if !strings.Contains(j, want) {
			t.Errorf("FULL JSON deveria conter %q\n%s", want, j)
		}
	}
}

func TestPatientResourceAnonymizedJSON(t *testing.T) {
	a := anonymize.New("salt")
	res := patientResource(a.Patient(rawJoao(), anonymize.LevelAnonymized))
	j, err := fhir.ToJSON(res)
	if err != nil {
		t.Fatal(err)
	}
	// Deve pseudonimizar e conter faixa etária.
	if !strings.Contains(j, `"hash`) || !strings.Contains(j, "age-range") {
		t.Errorf("ANONYMIZED JSON deveria ter id hash e extensão de faixa etária\n%s", j)
	}
	// NÃO deve vazar nome nem CPF nem cidade.
	for _, leak := range []string{"João", "111.111.111-11", "Brasilia", "birthDate"} {
		if strings.Contains(j, leak) {
			t.Errorf("ANONYMIZED JSON vazou %q\n%s", leak, j)
		}
	}
}

func TestObservationResourceValue(t *testing.T) {
	v := 8.1
	obs := observationResource("P000001", &pdpb.ClinicalEvent{
		EventId: "EVO01", EventType: "Observation", Code: "HbA1c",
		Description: "Hemoglobina glicada", EventDate: "2024-02-10", Value: &v, Unit: "%",
	})
	if obs.ValueQuantity == nil || obs.ValueQuantity.Value != 8.1 || obs.ValueQuantity.Unit != "%" {
		t.Errorf("valueQuantity incorreto: %+v", obs.ValueQuantity)
	}
	if obs.Subject == nil || obs.Subject.Reference != "Patient/P000001" {
		t.Errorf("subject incorreto: %+v", obs.Subject)
	}

	noVal := observationResource("P000001", &pdpb.ClinicalEvent{EventId: "X", EventType: "Observation"})
	if noVal.ValueQuantity != nil {
		t.Errorf("sem valor não deveria ter valueQuantity: %+v", noVal.ValueQuantity)
	}
}

func TestToPercentages(t *testing.T) {
	got := toPercentages([]*pdpb.Count{{Key: "male", Count: 6}, {Key: "female", Count: 7}}, 13)
	if len(got) != 2 {
		t.Fatalf("esperado 2, veio %d", len(got))
	}
	if got[0].Key != "male" || got[0].Count != 6 {
		t.Errorf("primeiro item incorreto: %+v", got[0])
	}
	// 6/13 = 46.15..., 7/13 = 53.84...
	if got[0].Percentage < 46.1 || got[0].Percentage > 46.2 {
		t.Errorf("percentual male = %v, quer ~46.15", got[0].Percentage)
	}
	if got[1].Percentage < 53.8 || got[1].Percentage > 53.9 {
		t.Errorf("percentual female = %v, quer ~53.85", got[1].Percentage)
	}
}
