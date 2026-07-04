// Package fhir define os structs dos recursos HL7/FHIR usados pelo projeto.
// Campos vazios usam omitempty, então dados removidos pela anonimização somem do JSON.
package fhir

import (
	"encoding/json"

	"google.golang.org/protobuf/types/known/structpb"
)

type HumanName struct {
	Text string `json:"text,omitempty"`
}

type Identifier struct {
	System string `json:"system,omitempty"`
	Value  string `json:"value,omitempty"`
}

type Address struct {
	City  string `json:"city,omitempty"`
	State string `json:"state,omitempty"`
}

type Extension struct {
	URL         string `json:"url"`
	ValueString string `json:"valueString,omitempty"`
}

type CodeableConcept struct {
	Text string `json:"text,omitempty"`
}

type Reference struct {
	Reference string `json:"reference,omitempty"`
}

type Quantity struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

type Period struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

type Patient struct {
	ResourceType string       `json:"resourceType"`
	ID           string       `json:"id,omitempty"`
	Identifier   []Identifier `json:"identifier,omitempty"`
	Name         []HumanName  `json:"name,omitempty"`
	Gender       string       `json:"gender,omitempty"`
	BirthDate    string       `json:"birthDate,omitempty"`
	Address      []Address    `json:"address,omitempty"`
	Extension    []Extension  `json:"extension,omitempty"`
}

type Encounter struct {
	ResourceType string            `json:"resourceType"`
	ID           string            `json:"id,omitempty"`
	Status       string            `json:"status,omitempty"`
	Class        *CodeableConcept  `json:"class,omitempty"`
	Type         []CodeableConcept `json:"type,omitempty"`
	ServiceType  *CodeableConcept  `json:"serviceType,omitempty"`
	Period       *Period           `json:"period,omitempty"`
	Subject      *Reference        `json:"subject,omitempty"`
}

type Condition struct {
	ResourceType string           `json:"resourceType"`
	ID           string           `json:"id,omitempty"`
	Code         *CodeableConcept `json:"code,omitempty"`
	Subject      *Reference       `json:"subject,omitempty"`
	RecordedDate string           `json:"recordedDate,omitempty"`
}

type Observation struct {
	ResourceType      string           `json:"resourceType"`
	ID                string           `json:"id,omitempty"`
	Status            string           `json:"status,omitempty"`
	Code              *CodeableConcept `json:"code,omitempty"`
	Subject           *Reference       `json:"subject,omitempty"`
	EffectiveDateTime string           `json:"effectiveDateTime,omitempty"`
	ValueQuantity     *Quantity        `json:"valueQuantity,omitempty"`
}

type Dosage struct {
	Text        string        `json:"text,omitempty"`
	DoseAndRate []DoseAndRate `json:"doseAndRate,omitempty"`
}

type DoseAndRate struct {
	DoseQuantity *Quantity `json:"doseQuantity,omitempty"`
}

type MedicationRequest struct {
	ResourceType              string           `json:"resourceType"`
	ID                        string           `json:"id,omitempty"`
	Status                    string           `json:"status,omitempty"`
	Intent                    string           `json:"intent,omitempty"`
	MedicationCodeableConcept *CodeableConcept `json:"medicationCodeableConcept,omitempty"`
	Subject                   *Reference       `json:"subject,omitempty"`
	AuthoredOn                string           `json:"authoredOn,omitempty"`
	DosageInstruction         []Dosage         `json:"dosageInstruction,omitempty"`
}

type ResearchStudy struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id,omitempty"`
	Title        string `json:"title,omitempty"`
	Status       string `json:"status,omitempty"`
}

type Bundle struct {
	ResourceType string  `json:"resourceType"`
	Type         string  `json:"type"`
	Total        int     `json:"total"`
	Entry        []Entry `json:"entry,omitempty"`
}

type Entry struct {
	Resource any `json:"resource"`
}

// NewBundle agrupa vários recursos num Bundle do tipo "collection".
func NewBundle(resources ...any) Bundle {
	entries := make([]Entry, 0, len(resources))
	for _, r := range resources {
		entries = append(entries, Entry{Resource: r})
	}
	return Bundle{ResourceType: "Bundle", Type: "collection", Total: len(resources), Entry: entries}
}

// ToJSON serializa um recurso ou Bundle em JSON indentado.
func ToJSON(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ToStruct converte um recurso ou Bundle FHIR em google.protobuf.Struct (objeto JSON).
func ToStruct(v any) (*structpb.Struct, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return structpb.NewStruct(m)
}
