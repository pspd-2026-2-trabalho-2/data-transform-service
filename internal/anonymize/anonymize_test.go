package anonymize

import (
	"strings"
	"testing"
)

func rawJoao() RawPatient {
	return RawPatient{
		ID:        "P000001",
		FullName:  "João da Silva",
		BirthDate: "1970-05-10",
		Gender:    "male",
		City:      "Brasilia",
		State:     "DF",
		CPF:       "111.111.111-11",
		CNS:       "700000000000001",
	}
}

func TestPatientFull(t *testing.T) {
	v := New("salt").Patient(rawJoao(), LevelFull)
	if v.ID != "P000001" || v.Name != "João da Silva" || v.CPF == "" || v.BirthDate != "1970-05-10" {
		t.Errorf("FULL deveria manter tudo: %+v", v)
	}
}

func TestPatientPartial(t *testing.T) {
	v := New("salt").Patient(rawJoao(), LevelPartial)
	if v.ID != "P000001" {
		t.Errorf("PARTIAL mantém o id real, veio %q", v.ID)
	}
	if v.Name != "J.S." {
		t.Errorf("PARTIAL deveria ter iniciais J.S., veio %q", v.Name)
	}
	if v.CPF != "" || v.CNS != "" {
		t.Errorf("PARTIAL deveria remover CPF/CNS, veio %q/%q", v.CPF, v.CNS)
	}
	if v.BirthDate != "1970" {
		t.Errorf("PARTIAL deveria expor só o ano, veio %q", v.BirthDate)
	}
	if v.City != "Brasilia" || v.State != "DF" {
		t.Errorf("PARTIAL mantém cidade/estado, veio %q/%q", v.City, v.State)
	}
}

func TestPatientAnonymized(t *testing.T) {
	v := New("salt").Patient(rawJoao(), LevelAnonymized)
	if !strings.HasPrefix(v.ID, "hash") {
		t.Errorf("ANONYMIZED deveria pseudonimizar o id, veio %q", v.ID)
	}
	if v.Name != "" || v.CPF != "" || v.CNS != "" || v.City != "" || v.BirthDate != "" {
		t.Errorf("ANONYMIZED deveria remover nome/CPF/CNS/cidade/data, veio %+v", v)
	}
	if v.AgeRange == "" {
		t.Error("ANONYMIZED deveria incluir faixa etária")
	}
	if v.State != "DF" {
		t.Errorf("ANONYMIZED mantém o estado, veio %q", v.State)
	}
}

func TestPseudonymStableAndSalted(t *testing.T) {
	a1 := New("salt-a")
	if a1.Pseudonym("P000001") != a1.Pseudonym("P000001") {
		t.Error("pseudônimo deveria ser determinístico para o mesmo id")
	}
	a2 := New("salt-b")
	if a1.Pseudonym("P000001") == a2.Pseudonym("P000001") {
		t.Error("salts diferentes deveriam gerar pseudônimos diferentes")
	}
}

func TestInitials(t *testing.T) {
	cases := map[string]string{
		"João da Silva": "J.S.",
		"Maria Souza":   "M.S.",
		"Ana":           "A.",
		"":              "",
	}
	for in, want := range cases {
		if got := initials(in); got != want {
			t.Errorf("initials(%q) = %q, quer %q", in, got, want)
		}
	}
}

func TestBirthYear(t *testing.T) {
	if got := birthYear("1970-05-10"); got != "1970" {
		t.Errorf("birthYear = %q", got)
	}
	if got := birthYear(""); got != "" {
		t.Errorf("birthYear vazio = %q", got)
	}
}
