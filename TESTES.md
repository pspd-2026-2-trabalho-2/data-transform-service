# Guia de Testes — data-transform-service

Como subir o serviço e validar a **anonimização por nível** e a **conversão HL7/FHIR** **pelo
Postman** (gRPC). O serviço é um transformador puro: os dados vão **no request** — não precisa de
banco nem de outro serviço no ar.

---

## 1. Pré-requisitos
- **Go 1.25+** (e opcionalmente **Docker**)
- **Postman** (com suporte a gRPC)

## 2. Subir o serviço
```bash
cp .env.example .env
go run ./cmd/server
```
(ou `docker compose up -d --build`)

Logs esperados: `servidor gRPC ouvindo port=50053`.
- gRPC: **localhost:50053** · Métricas/health: **http://localhost:9091**

## 3. Testes automatizados (opcional, mas recomendado)
```bash
go test ./...   # regras de anonimização + conversão FHIR (sem rede)
```

---

## 4. Configurar o Postman (uma vez)
1. **New → gRPC Request**.
2. Em **Enter server URL**: `localhost:50053` (deixe **TLS desligado**).
3. O serviço tem **server reflection**: o Postman lista os métodos sozinho no dropdown **Select a method**.
   (Alternativa: **Import a .proto file** → `proto/datatransform/v1/datatransform.proto`.)
4. Escolha o método, cole o JSON na aba **Message** e clique **Invoke**.

> Métodos sob `datatransform.v1.DataTransformService`. A resposta traz o FHIR no campo `fhirJson`
> (uma string com o JSON FHIR dentro — é esperado vir "escapado").

---

## 5. Casos de teste

O destaque (CT-01 a CT-03) é ver o **mesmo paciente encolher** conforme o nível cai.

### CT-01 — TransformPatient (FULL)
Esperado: Patient com nome completo, CPF/CNS, data exata, cidade/estado.
```json
{
  "patient": {
    "patientId": "P000001",
    "fullName": "João da Silva",
    "birthDate": "1970-05-10",
    "gender": "male",
    "city": "Brasilia",
    "state": "DF",
    "cpf": "111.111.111-11",
    "cns": "700000000000001"
  },
  "accessLevel": "FULL"
}
```

### CT-02 — TransformPatient (PARTIAL)
Esperado: nome `J.S.`, `birthDate: "1970"`, **sem** CPF/CNS; mantém cidade/estado.
```json
{
  "patient": {
    "patientId": "P000001",
    "fullName": "João da Silva",
    "birthDate": "1970-05-10",
    "gender": "male",
    "city": "Brasilia",
    "state": "DF",
    "cpf": "111.111.111-11",
    "cns": "700000000000001"
  },
  "accessLevel": "PARTIAL"
}
```

### CT-03 — TransformPatient (ANONYMIZED)
Esperado: `id: "hash484845"`, **sem** nome/CPF/cidade/data; faixa etária `40-59`; só o estado.
```json
{
  "patient": {
    "patientId": "P000001",
    "fullName": "João da Silva",
    "birthDate": "1970-05-10",
    "gender": "male",
    "city": "Brasilia",
    "state": "DF",
    "cpf": "111.111.111-11",
    "cns": "700000000000001"
  },
  "accessLevel": "ANONYMIZED"
}
```

### CT-04 — TransformClinicalSummary (FULL → Bundle)
Esperado: Bundle com Patient + Encounter + 2 Condition + 2 Observation + 2 MedicationRequest.
```json
{
  "summary": {
    "patient": {
      "patientId": "P000001", "fullName": "João da Silva", "birthDate": "1970-05-10",
      "gender": "male", "city": "Brasilia", "state": "DF",
      "cpf": "111.111.111-11", "cns": "700000000000001"
    },
    "lastEncounter": {
      "encounterId": "ENC16", "patientId": "P000001", "startDate": "2024-08-01",
      "endDate": "2024-08-01", "encounterType": "Retorno", "department": "Cardiologia"
    },
    "conditions": [
      { "eventId": "EVC01", "patientId": "P000001", "eventType": "Condition", "code": "Diabetes", "description": "Diabetes Mellitus Tipo 2", "eventDate": "2024-02-10" },
      { "eventId": "EVC14", "patientId": "P000001", "eventType": "Condition", "code": "Hipertensao", "description": "Hipertensão Arterial", "eventDate": "2024-08-01" }
    ],
    "recentObservations": [
      { "eventId": "EVO01", "patientId": "P000001", "eventType": "Observation", "code": "HbA1c", "description": "Hemoglobina glicada", "eventDate": "2024-02-10", "value": 8.1, "unit": "%" },
      { "eventId": "EVO14", "patientId": "P000001", "eventType": "Observation", "code": "Glicemia", "description": "Glicemia de jejum", "eventDate": "2024-02-10", "value": 182, "unit": "mg/dL" }
    ],
    "activeMedications": [
      { "eventId": "EVM01", "patientId": "P000001", "eventType": "Medication", "code": "Metformina", "description": "Metformina 850 mg", "eventDate": "2024-02-10", "value": 850, "unit": "mg" },
      { "eventId": "EVM15", "patientId": "P000001", "eventType": "Medication", "code": "Losartana", "description": "Losartana 50 mg", "eventDate": "2024-08-01", "value": 50, "unit": "mg" }
    ]
  },
  "accessLevel": "FULL"
}
```

### CT-05 — TransformClinicalHistory (ANONYMIZED → Bundle)
Esperado: Bundle com Patient (id `hash484845`) + 4 eventos, todos referenciando `Patient/hash484845`.
```json
{
  "patient": {
    "patientId": "P000001", "fullName": "João da Silva", "birthDate": "1970-05-10",
    "gender": "male", "city": "Brasilia", "state": "DF",
    "cpf": "111.111.111-11", "cns": "700000000000001"
  },
  "events": [
    { "eventId": "EVC01", "patientId": "P000001", "eventType": "Condition", "code": "Diabetes", "description": "Diabetes Mellitus Tipo 2", "eventDate": "2024-02-10" },
    { "eventId": "EVO01", "patientId": "P000001", "eventType": "Observation", "code": "HbA1c", "description": "Hemoglobina glicada", "eventDate": "2024-02-10", "value": 8.1, "unit": "%" },
    { "eventId": "EVM01", "patientId": "P000001", "eventType": "Medication", "code": "Metformina", "description": "Metformina 850 mg", "eventDate": "2024-02-10", "value": 850, "unit": "mg" },
    { "eventId": "EVC14", "patientId": "P000001", "eventType": "Condition", "code": "Hipertensao", "description": "Hipertensão Arterial", "eventDate": "2024-08-01" }
  ],
  "accessLevel": "ANONYMIZED"
}
```

### CT-06 — TransformPatientList (PARTIAL → Bundle)
Esperado: Bundle com 2 Patient; nomes `J.S.` e `M.S.`, sem CPF/CNS.
```json
{
  "patients": [
    { "patientId": "P000001", "fullName": "João da Silva", "birthDate": "1970-05-10", "gender": "male", "city": "Brasilia", "state": "DF", "cpf": "111.111.111-11", "cns": "700000000000001" },
    { "patientId": "P000002", "fullName": "Maria Souza", "birthDate": "1985-03-22", "gender": "female", "city": "Goiania", "state": "GO", "cpf": "222.222.222-22", "cns": "700000000000002" }
  ],
  "accessLevel": "PARTIAL"
}
```

### CT-07 — TransformCohortExams (ANONYMIZED → Bundle)
Esperado: Bundle com 2 Patient pseudonimizados (ids distintos) + suas Observations; nenhum nome/CPF.
```json
{
  "patients": [
    {
      "patient": { "patientId": "P000001", "fullName": "João da Silva", "birthDate": "1970-05-10", "gender": "male", "city": "Brasilia", "state": "DF", "cpf": "111.111.111-11", "cns": "700000000000001" },
      "exams": [
        { "eventId": "EVO01", "patientId": "P000001", "eventType": "Observation", "code": "HbA1c", "description": "Hemoglobina glicada", "eventDate": "2024-02-10", "value": 8.1, "unit": "%" },
        { "eventId": "EVO14", "patientId": "P000001", "eventType": "Observation", "code": "Glicemia", "description": "Glicemia de jejum", "eventDate": "2024-02-10", "value": 182, "unit": "mg/dL" }
      ]
    },
    {
      "patient": { "patientId": "P000002", "fullName": "Maria Souza", "birthDate": "1985-03-22", "gender": "female", "city": "Goiania", "state": "GO", "cpf": "222.222.222-22", "cns": "700000000000002" },
      "exams": [
        { "eventId": "EVO02", "patientId": "P000002", "eventType": "Observation", "code": "HbA1c", "description": "Hemoglobina glicada", "eventDate": "2024-03-05", "value": 7.2, "unit": "%" }
      ]
    }
  ],
  "accessLevel": "ANONYMIZED"
}
```

### CT-08 — TransformCohortStatistics (percentuais)
Esperado: F 53.85% / M 46.15%; faixas 23.08/46.15/30.77%; Metformina 76.92%, Insulina 30.77%, Losartana 15.38%.
```json
{
  "stats": {
    "conditionCode": "Diabetes",
    "totalPatients": 13,
    "bySex": [ { "key": "female", "count": 7 }, { "key": "male", "count": 6 } ],
    "byAgeRange": [ { "key": "18-39", "count": 3 }, { "key": "40-59", "count": 6 }, { "key": "60+", "count": 4 } ],
    "meanHba1c": 7.76,
    "medianHba1c": 7.6,
    "medicationFrequency": [ { "key": "Metformina", "count": 10 }, { "key": "Insulina", "count": 4 }, { "key": "Losartana", "count": 2 } ]
  }
}
```

### CT-09 — TransformProjects (ResearchStudy)
Esperado: Bundle com 2 ResearchStudy: PRJ01 `active`, PRJ02 `completed`.
```json
{
  "projects": [
    { "projectId": "PRJ01", "title": "Coorte Diabetes Tipo 2", "researcherUsername": "pesq.lima", "conditionCode": "Diabetes", "status": "Aprovado", "validUntil": "2027-12-31" },
    { "projectId": "PRJ02", "title": "Coorte Hipertensão Resistente", "researcherUsername": "pesq.lima", "conditionCode": "Hipertensao", "status": "Expirado", "validUntil": "2024-01-01" }
  ]
}
```

---

## 6. Observabilidade (Prometheus)
Depois de rodar alguns casos:
- `http://localhost:9091/healthz` → `ok`
- `http://localhost:9091/metrics` → procure por `fhir_transforms_total`, `grpc_server_handled_total`,
  `grpc_server_handling_seconds`, `go_goroutines`, `process_resident_memory_bytes`.

## 7. Encerrar
Pare o serviço com `Ctrl+C` (ou `docker compose down`).

## Troubleshooting
| Sintoma | Solução |
|---|---|
| Postman não conecta | serviço fora do ar ou porta errada (gRPC é 50053); TLS desligado |
| `INVALID_ARGUMENT` | faltou campo obrigatório no payload (ex.: `patient`, `summary`, `stats`) |
| `fhirJson` vem "escapado" | é esperado: é uma string com o JSON FHIR dentro |
| métodos não aparecem no Postman | a reflection precisa do serviço no ar; recarregue após conectar, ou importe o `.proto` |
