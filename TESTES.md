# Guia de Testes — data-transform-service

Como subir o serviço e validar a **anonimização por nível** e a **conversão HL7/FHIR** **pelo
Postman** (gRPC). O serviço é um transformador puro: os dados vão **no request** — não precisa de
banco nem de outro serviço no ar.

> No Postman (gRPC) os campos usam os **nomes do `.proto` (snake_case)**, ex.: `patient_id`,
> `access_level`, `full_name`. A resposta FHIR é o próprio objeto (recurso ou Bundle), com os nomes
> padrão FHIR (`resourceType`, `birthDate`, ...).
>
> Os **valores** de `event_type`, códigos e status seguem a convenção do banco real
> (MAIÚSCULO/inglês): `CONDITION`/`OBSERVATION`/`MEDICATION`, `DIABETES`, `HBA1C`, `METFORMIN`,
> `ENDOCRINOLOGY`, `APPROVED`, etc.

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

> Métodos sob `datatransform.v1.DataTransformService`. As respostas FHIR já vêm como **objeto JSON**
> (um recurso, ex.: `Patient`, ou um `Bundle`) — prontas para copiar e validar no validator.fhir.org.

---

## 5. Casos de teste

O destaque (CT-01 a CT-03) é ver o **mesmo paciente encolher** conforme o nível cai.

### CT-01 — TransformPatient (FULL)
Esperado: Patient com nome completo, CPF/CNS, data exata, cidade/estado.
```json
{
  "patient": {
    "patient_id": "P000001",
    "full_name": "João da Silva",
    "birth_date": "1970-05-10",
    "gender": "male",
    "city": "Brasilia",
    "state": "DF",
    "cpf": "111.111.111-11",
    "cns": "700000000000001"
  },
  "access_level": "FULL"
}
```

### CT-02 — TransformPatient (PARTIAL)
Esperado: nome `J.S.`, `birthDate: "1970"`, **sem** CPF/CNS; mantém cidade/estado.
```json
{
  "patient": {
    "patient_id": "P000001",
    "full_name": "João da Silva",
    "birth_date": "1970-05-10",
    "gender": "male",
    "city": "Brasilia",
    "state": "DF",
    "cpf": "111.111.111-11",
    "cns": "700000000000001"
  },
  "access_level": "PARTIAL"
}
```

### CT-03 — TransformPatient (ANONYMIZED)
Esperado: `id: "hash484845"`, **sem** nome/CPF/cidade/data; faixa etária `40-59`; só o estado.
```json
{
  "patient": {
    "patient_id": "P000001",
    "full_name": "João da Silva",
    "birth_date": "1970-05-10",
    "gender": "male",
    "city": "Brasilia",
    "state": "DF",
    "cpf": "111.111.111-11",
    "cns": "700000000000001"
  },
  "access_level": "ANONYMIZED"
}
```

### CT-04 — TransformClinicalSummary (FULL → Bundle)
Esperado: Bundle com Patient + Encounter + 2 Condition + 2 Observation + 2 MedicationRequest.
```json
{
  "summary": {
    "patient": {
      "patient_id": "P000001", "full_name": "João da Silva", "birth_date": "1970-05-10",
      "gender": "male", "city": "Brasilia", "state": "DF",
      "cpf": "111.111.111-11", "cns": "700000000000001"
    },
    "last_encounter": {
      "encounter_id": "ENC16", "patient_id": "P000001", "start_date": "2024-08-01",
      "end_date": "2024-08-01", "encounter_type": "Retorno", "department": "Cardiologia"
    },
    "conditions": [
      { "event_id": "EVC01", "patient_id": "P000001", "event_type": "CONDITION", "code": "DIABETES", "description": "Diabetes Mellitus Tipo 2", "event_date": "2024-02-10" },
      { "event_id": "EVC14", "patient_id": "P000001", "event_type": "CONDITION", "code": "HYPERTENSION", "description": "Hipertensão Arterial", "event_date": "2024-08-01" }
    ],
    "recent_observations": [
      { "event_id": "EVO01", "patient_id": "P000001", "event_type": "OBSERVATION", "code": "HBA1C", "description": "Hemoglobina glicada", "event_date": "2024-02-10", "value": 8.1, "unit": "%" },
      { "event_id": "EVO14", "patient_id": "P000001", "event_type": "OBSERVATION", "code": "GLUCOSE", "description": "Glicemia de jejum", "event_date": "2024-02-10", "value": 182, "unit": "mg/dL" }
    ],
    "active_medications": [
      { "event_id": "EVM01", "patient_id": "P000001", "event_type": "MEDICATION", "code": "METFORMIN", "description": "Metformina 850 mg", "event_date": "2024-02-10", "value": 850, "unit": "mg" },
      { "event_id": "EVM15", "patient_id": "P000001", "event_type": "MEDICATION", "code": "LOSARTAN", "description": "Losartana 50 mg", "event_date": "2024-08-01", "value": 50, "unit": "mg" }
    ]
  },
  "access_level": "FULL"
}
```

### CT-05 — TransformClinicalHistory (ANONYMIZED → Bundle)
Esperado: Bundle com Patient (id `hash484845`) + 4 eventos, todos referenciando `Patient/hash484845`.
```json
{
  "patient": {
    "patient_id": "P000001", "full_name": "João da Silva", "birth_date": "1970-05-10",
    "gender": "male", "city": "Brasilia", "state": "DF",
    "cpf": "111.111.111-11", "cns": "700000000000001"
  },
  "events": [
    { "event_id": "EVC01", "patient_id": "P000001", "event_type": "CONDITION", "code": "DIABETES", "description": "Diabetes Mellitus Tipo 2", "event_date": "2024-02-10" },
    { "event_id": "EVO01", "patient_id": "P000001", "event_type": "OBSERVATION", "code": "HBA1C", "description": "Hemoglobina glicada", "event_date": "2024-02-10", "value": 8.1, "unit": "%" },
    { "event_id": "EVM01", "patient_id": "P000001", "event_type": "MEDICATION", "code": "METFORMIN", "description": "Metformina 850 mg", "event_date": "2024-02-10", "value": 850, "unit": "mg" },
    { "event_id": "EVC14", "patient_id": "P000001", "event_type": "CONDITION", "code": "HYPERTENSION", "description": "Hipertensão Arterial", "event_date": "2024-08-01" }
  ],
  "access_level": "ANONYMIZED"
}
```

### CT-06 — TransformPatientList (PARTIAL → Bundle)
Esperado: Bundle com 2 Patient; nomes `J.S.` e `M.S.`, sem CPF/CNS.
```json
{
  "patients": [
    { "patient_id": "P000001", "full_name": "João da Silva", "birth_date": "1970-05-10", "gender": "male", "city": "Brasilia", "state": "DF", "cpf": "111.111.111-11", "cns": "700000000000001" },
    { "patient_id": "P000002", "full_name": "Maria Souza", "birth_date": "1985-03-22", "gender": "female", "city": "Goiania", "state": "GO", "cpf": "222.222.222-22", "cns": "700000000000002" }
  ],
  "access_level": "PARTIAL"
}
```

### CT-07 — TransformCohortExams (ANONYMIZED → Bundle)
Esperado: Bundle com 2 Patient pseudonimizados (ids distintos) + suas Observations; nenhum nome/CPF.
```json
{
  "patients": [
    {
      "patient": { "patient_id": "P000001", "full_name": "João da Silva", "birth_date": "1970-05-10", "gender": "male", "city": "Brasilia", "state": "DF", "cpf": "111.111.111-11", "cns": "700000000000001" },
      "exams": [
        { "event_id": "EVO01", "patient_id": "P000001", "event_type": "OBSERVATION", "code": "HBA1C", "description": "Hemoglobina glicada", "event_date": "2024-02-10", "value": 8.1, "unit": "%" },
        { "event_id": "EVO14", "patient_id": "P000001", "event_type": "OBSERVATION", "code": "GLUCOSE", "description": "Glicemia de jejum", "event_date": "2024-02-10", "value": 182, "unit": "mg/dL" }
      ]
    },
    {
      "patient": { "patient_id": "P000002", "full_name": "Maria Souza", "birth_date": "1985-03-22", "gender": "female", "city": "Goiania", "state": "GO", "cpf": "222.222.222-22", "cns": "700000000000002" },
      "exams": [
        { "event_id": "EVO02", "patient_id": "P000002", "event_type": "OBSERVATION", "code": "HBA1C", "description": "Hemoglobina glicada", "event_date": "2024-03-05", "value": 7.2, "unit": "%" }
      ]
    }
  ],
  "access_level": "ANONYMIZED"
}
```

### CT-08 — TransformCohortStatistics (percentuais)
Esperado: sexo F 53.85% / M 46.15%; faixas 23.08/46.15/30.77%; medicamentos METFORMIN 76.92%,
INSULIN 30.77%, LOSARTAN 15.38%; departamentos ENDOCRINOLOGY 100%, CARDIOLOGY 15.38%.
```json
{
  "stats": {
    "condition_code": "DIABETES",
    "total_patients": 13,
    "by_sex": [ { "key": "female", "count": 7 }, { "key": "male", "count": 6 } ],
    "by_age_range": [ { "key": "18-39", "count": 3 }, { "key": "40-59", "count": 6 }, { "key": "60+", "count": 4 } ],
    "mean_hba1c": 7.76,
    "median_hba1c": 7.6,
    "medication_frequency": [ { "key": "METFORMIN", "count": 10 }, { "key": "INSULIN", "count": 4 }, { "key": "LOSARTAN", "count": 2 } ],
    "by_department": [ { "key": "ENDOCRINOLOGY", "count": 13 }, { "key": "CARDIOLOGY", "count": 2 } ]
  }
}
```

### CT-09 — TransformProjects (ResearchStudy)
Esperado: Bundle com 2 ResearchStudy: PRJ01 `active`, PRJ02 `completed`.
```json
{
  "projects": [
    { "project_id": "PRJ01", "title": "Coorte Diabetes Tipo 2", "researcher_username": "pesq.lima", "condition_code": "DIABETES", "status": "APPROVED", "valid_until": "2027-12-31" },
    { "project_id": "PRJ02", "title": "Coorte Hipertensão Resistente", "researcher_username": "pesq.lima", "condition_code": "HYPERTENSION", "status": "EXPIRED", "valid_until": "2024-01-01" }
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
| `INVALID_ARGUMENT` (ex.: `patient é obrigatório`) | campo vazio; use os nomes **snake_case** (`patient`, `access_level`, `patient_id`) e preencha os valores |
| Postman não conecta | serviço fora do ar ou porta errada (gRPC é 50053); TLS desligado |
| métodos não aparecem no Postman | a reflection precisa do serviço no ar; recarregue após conectar, ou importe o `.proto` |
