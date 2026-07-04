// Package anonymize aplica as regras de nível de acesso (supressão, generalização, pseudonimização).
package anonymize

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// Level é o nível de acesso aplicado.
type Level int

const (
	LevelFull Level = iota
	LevelPartial
	LevelAnonymized
	LevelAggregated
)

// RawPatient são os dados crus de um paciente.
type RawPatient struct {
	ID        string
	FullName  string
	BirthDate string // ISO 8601 (YYYY-MM-DD)
	Gender    string
	City      string
	State     string
	CPF       string
	CNS       string
}

// PatientView são os campos já filtrados/mascarados prontos para virar FHIR.
type PatientView struct {
	ID        string
	Name      string
	BirthDate string
	AgeRange  string // preenchido no ANONYMIZED (data exata suprimida)
	Gender    string
	City      string
	State     string
	CPF       string
	CNS       string
}

// Anonymizer aplica as regras; guarda o salt da pseudonimização.
type Anonymizer struct {
	salt string
}

func New(salt string) *Anonymizer { return &Anonymizer{salt: salt} }

// Patient aplica o nível de acesso ao paciente.
func (a *Anonymizer) Patient(r RawPatient, level Level) PatientView {
	switch level {
	case LevelPartial:
		return PatientView{
			ID:        r.ID,
			Name:      initials(r.FullName),
			BirthDate: birthYear(r.BirthDate), // só o ano
			Gender:    r.Gender,
			City:      r.City,
			State:     r.State,
			// CPF/CNS removidos
		}
	case LevelAnonymized:
		return PatientView{
			ID:       a.Pseudonym(r.ID),
			AgeRange: ageRange(r.BirthDate), // faixa etária no lugar da data
			Gender:   r.Gender,
			State:    r.State,
			// nome, CPF, CNS, cidade e data exata removidos
		}
	default: // LevelFull
		return PatientView{
			ID:        r.ID,
			Name:      r.FullName,
			BirthDate: r.BirthDate,
			Gender:    r.Gender,
			City:      r.City,
			State:     r.State,
			CPF:       r.CPF,
			CNS:       r.CNS,
		}
	}
}

// Pseudonym gera um identificador estável e não reversível para o paciente.
func (a *Anonymizer) Pseudonym(id string) string {
	sum := sha256.Sum256([]byte(a.salt + id))
	return "hash" + hex.EncodeToString(sum[:])[:6]
}

// initials transforma "João da Silva" em "J.S.".
func initials(fullName string) string {
	var letters []string
	for _, word := range strings.Fields(fullName) {
		if len(word) <= 2 { // ignora conectivos: da, de, do, e
			continue
		}
		r := []rune(word)
		letters = append(letters, strings.ToUpper(string(r[0])))
	}
	if len(letters) == 0 {
		return ""
	}
	return strings.Join(letters, ".") + "."
}

// birthYear extrai o ano de uma data ISO.
func birthYear(birthDate string) string {
	if len(birthDate) >= 4 {
		return birthDate[:4]
	}
	return ""
}

// ageRange classifica a idade em faixas.
func ageRange(birthDate string) string {
	t, err := time.Parse("2006-01-02", birthDate)
	if err != nil {
		return ""
	}
	age := yearsSince(t)
	switch {
	case age <= 17:
		return "0-17"
	case age <= 39:
		return "18-39"
	case age <= 59:
		return "40-59"
	default:
		return "60+"
	}
}

func yearsSince(t time.Time) int {
	now := time.Now()
	years := now.Year() - t.Year()
	if now.YearDay() < t.YearDay() {
		years--
	}
	return years
}

// String devolve o nome textual do nível (para logs/labels).
func (l Level) String() string {
	switch l {
	case LevelPartial:
		return "PARTIAL"
	case LevelAnonymized:
		return "ANONYMIZED"
	case LevelAggregated:
		return "AGGREGATED"
	default:
		return "FULL"
	}
}
