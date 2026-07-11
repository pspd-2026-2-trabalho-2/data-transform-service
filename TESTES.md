# Guia de Testes — data-transform-service

Como subir o serviço e validar a **anonimização por nível** e a **conversão HL7/FHIR** **pelo
Postman** (gRPC). O serviço é um transformador puro: os dados vão **no request** — não precisa de
banco nem de outro serviço no ar.

> No Postman (gRPC) os campos usam os **nomes do `.proto` (snake_case)**, ex.: `patient_id`,
> `access_level`, `full_name`. A resposta FHIR é o próprio objeto (recurso ou Bundle), com os nomes
> padrão FHIR (`resourceType`, `birthDate`, ...).
>
> Os exemplos usam um **paciente real do banco do professor** — **Ana Almeida** (`P030000001`), com
> seus **eventos clínicos reais** (Insuficiência Cardíaca, IMC, Sinvastatina) — e a **coorte real de
> Diabetes** (30.110 pacientes). Os **valores** de `event_type`, códigos e status seguem a convenção
> do banco real (MAIÚSCULO/inglês): `CONDITION`/`OBSERVATION`/`MEDICATION`, `HEART_FAILURE`, `BMI`,
> `SIMVASTATIN`, `PEDIATRICS`, `APPROVED`, etc.

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

O destaque (CT-01 a CT-03) é ver o **mesmo paciente real encolher** conforme o nível cai.

### CT-01 — TransformPatient (FULL)
Esperado: Patient com nome completo, CPF/CNS, data exata, cidade/estado.
```json
{
  "patient": {
    "patient_id": "P030000001",
    "full_name": "Ana Almeida",
    "birth_date": "2009-03-12",
    "gender": "female",
    "city": "Formosa",
    "state": "GO",
    "cpf": "03000000001",
    "cns": "898030000000001"
  },
  "access_level": "FULL"
}
```

### CT-02 — TransformPatient (PARTIAL)
Esperado: nome `A.A.`, `birthDate: "2009"`, **sem** CPF/CNS; mantém cidade/estado (Formosa/GO).
```json
{
  "patient": {
    "patient_id": "P030000001",
    "full_name": "Ana Almeida",
    "birth_date": "2009-03-12",
    "gender": "female",
    "city": "Formosa",
    "state": "GO",
    "cpf": "03000000001",
    "cns": "898030000000001"
  },
  "access_level": "PARTIAL"
}
```

### CT-03 — TransformPatient (ANONYMIZED)
Esperado: `id: "hash8778a4"` (pseudônimo; o valor exato depende do `PSEUDONYM_SALT`), **sem**
nome/CPF/CNS/cidade/data; faixa etária `0-17`; só o estado (GO).
```json
{
  "patient": {
    "patient_id": "P030000001",
    "full_name": "Ana Almeida",
    "birth_date": "2009-03-12",
    "gender": "female",
    "city": "Formosa",
    "state": "GO",
    "cpf": "03000000001",
    "cns": "898030000000001"
  },
  "access_level": "ANONYMIZED"
}
```

### CT-04 — TransformClinicalSummary (FULL → Bundle)
Esperado: Bundle com Patient + Encounter + 1 Condition + 1 Observation + 1 MedicationRequest.
Dados **reais** da Ana (`GetClinicalSummary`): Insuficiência Cardíaca, IMC 26,9 e Sinvastatina, num
atendimento de internação (PEDIATRICS).
```json
{
  "summary": {
    "patient": {
      "patient_id": "P030000001", "full_name": "Ana Almeida", "birth_date": "2009-03-12",
      "gender": "female", "city": "Formosa", "state": "GO",
      "cpf": "03000000001", "cns": "898030000000001"
    },
    "last_encounter": {
      "encounter_id": "E03000000001", "patient_id": "P030000001", "start_date": "2024-08-05",
      "end_date": "2024-08-05", "encounter_type": "INPATIENT", "department": "PEDIATRICS"
    },
    "conditions": [
      { "event_id": "CE030000000001", "patient_id": "P030000001", "encounter_id": "E03000000001", "event_type": "CONDITION", "code": "HEART_FAILURE", "description": "Insuficiência Cardíaca", "event_date": "2024-08-05" }
    ],
    "recent_observations": [
      { "event_id": "CE030000000002", "patient_id": "P030000001", "encounter_id": "E03000000001", "event_type": "OBSERVATION", "code": "BMI", "description": "Índice de Massa Corporal", "event_date": "2024-08-05", "value": 26.9, "unit": "kg/m²" }
    ],
    "active_medications": [
      { "event_id": "CE030000000003", "patient_id": "P030000001", "encounter_id": "E03000000001", "event_type": "MEDICATION", "code": "SIMVASTATIN", "description": "Sinvastatina", "event_date": "2024-08-05", "value": 20, "unit": "mg" }
    ]
  },
  "access_level": "FULL"
}
```

### CT-05 — TransformClinicalHistory (ANONYMIZED → Bundle)
Esperado: Bundle com Patient (id `hash8778a4`) + 3 eventos reais, todos referenciando
`Patient/hash8778a4`.
```json
{
  "patient": {
    "patient_id": "P030000001", "full_name": "Ana Almeida", "birth_date": "2009-03-12",
    "gender": "female", "city": "Formosa", "state": "GO",
    "cpf": "03000000001", "cns": "898030000000001"
  },
  "events": [
    { "event_id": "CE030000000001", "patient_id": "P030000001", "encounter_id": "E03000000001", "event_type": "CONDITION", "code": "HEART_FAILURE", "description": "Insuficiência Cardíaca", "event_date": "2024-08-05" },
    { "event_id": "CE030000000002", "patient_id": "P030000001", "encounter_id": "E03000000001", "event_type": "OBSERVATION", "code": "BMI", "description": "Índice de Massa Corporal", "event_date": "2024-08-05", "value": 26.9, "unit": "kg/m²" },
    { "event_id": "CE030000000003", "patient_id": "P030000001", "encounter_id": "E03000000001", "event_type": "MEDICATION", "code": "SIMVASTATIN", "description": "Sinvastatina", "event_date": "2024-08-05", "value": 20, "unit": "mg" }
  ],
  "access_level": "ANONYMIZED"
}
```

### CT-06 — TransformPatientList (PARTIAL → Bundle)
Esperado: Bundle com 2 Patient; nomes em iniciais (`A.A.` e `M.S.`), sem CPF/CNS.
> O segundo paciente é **ilustrativo** (para mostrar a lista com mais de um).
```json
{
  "patients": [
    { "patient_id": "P030000001", "full_name": "Ana Almeida", "birth_date": "2009-03-12", "gender": "female", "city": "Formosa", "state": "GO", "cpf": "03000000001", "cns": "898030000000001" },
    { "patient_id": "PEX000002", "full_name": "Maria Souza", "birth_date": "1985-03-22", "gender": "female", "city": "Goiania", "state": "GO", "cpf": "222.222.222-22", "cns": "700000000000002" }
  ],
  "access_level": "PARTIAL"
}
```

### CT-07 — TransformCohortExams (ANONYMIZED → Bundle)
Esperado: Bundle com 2 Patient pseudonimizados (ids distintos) + suas Observations; nenhum nome/CPF.
> Exame da Ana é real (IMC); o segundo paciente é **ilustrativo**.
```json
{
  "patients": [
    {
      "patient": { "patient_id": "P030000001", "full_name": "Ana Almeida", "birth_date": "2009-03-12", "gender": "female", "city": "Formosa", "state": "GO", "cpf": "03000000001", "cns": "898030000000001" },
      "exams": [
        { "event_id": "CE030000000002", "patient_id": "P030000001", "encounter_id": "E03000000001", "event_type": "OBSERVATION", "code": "BMI", "description": "Índice de Massa Corporal", "event_date": "2024-08-05", "value": 26.9, "unit": "kg/m²" }
      ]
    },
    {
      "patient": { "patient_id": "PEX000002", "full_name": "Maria Souza", "birth_date": "1985-03-22", "gender": "female", "city": "Goiania", "state": "GO", "cpf": "222.222.222-22", "cns": "700000000000002" },
      "exams": [
        { "event_id": "EVO0300003", "patient_id": "PEX000002", "event_type": "OBSERVATION", "code": "HBA1C", "description": "Hemoglobina glicada", "event_date": "2025-03-05", "value": 7.2, "unit": "%" }
      ]
    }
  ],
  "access_level": "ANONYMIZED"
}
```

### CT-08 — TransformCohortStatistics (percentuais — coorte real)
Entrada: as contagens reais da coorte de Diabetes (`GetCohortStatistics` do patient-data).
Esperado (percentuais sobre 30.110): sexo F **49,9%** / M **50,1%**; faixas 0-17 **15,3%**, 18-39
**27,0%**, 40-59 **24,5%**, 60+ **33,2%**; medicamentos ~**37%** cada; departamentos ~**16–17%**
cada; HbA1c média **8,50** / mediana **8,5** (repassadas).
```json
{
  "stats": {
    "condition_code": "DIABETES",
    "total_patients": 30110,
    "by_sex": [ { "key": "female", "count": 15022 }, { "key": "male", "count": 15088 } ],
    "by_age_range": [ { "key": "0-17", "count": 4613 }, { "key": "18-39", "count": 8145 }, { "key": "40-59", "count": 7362 }, { "key": "60+", "count": 9990 } ],
    "mean_hba1c": 8.495,
    "median_hba1c": 8.5,
    "medication_frequency": [ { "key": "LOSARTAN", "count": 11362 }, { "key": "ENALAPRIL", "count": 11354 }, { "key": "SIMVASTATIN", "count": 11206 }, { "key": "METFORMIN", "count": 11139 }, { "key": "INSULIN", "count": 11084 } ],
    "by_department": [ { "key": "PEDIATRICS", "count": 5107 }, { "key": "NEPHROLOGY", "count": 5078 }, { "key": "INTERNAL_MEDICINE", "count": 5069 }, { "key": "INFECTIOUS_DISEASES", "count": 5043 }, { "key": "ICU", "count": 5021 }, { "key": "SURGERY", "count": 5019 }, { "key": "CARDIOLOGY", "count": 5015 }, { "key": "EMERGENCY", "count": 4964 }, { "key": "ORTHOPEDICS", "count": 4964 }, { "key": "ENDOCRINOLOGY", "count": 4956 }, { "key": "ONCOLOGY", "count": 4954 }, { "key": "TELEMEDICINE", "count": 4918 }, { "key": "PULMONOLOGY", "count": 4900 }, { "key": "GERIATRICS", "count": 4876 } ]
  }
}
```

### CT-09 — TransformProjects (ResearchStudy — projetos reais)
Esperado: Bundle com 3 ResearchStudy: PRJ01_G03 `active` (APPROVED), PRJ04_G03 `in-progress`
(PENDING) e PRJ05_G03 `completed` (EXPIRED). *(Os `title` são rótulos ilustrativos.)*
```json
{
  "projects": [
    { "project_id": "PRJ01_G03", "title": "Coorte de Diabetes", "researcher_username": "pes.mendes", "condition_code": "DIABETES", "status": "APPROVED", "valid_until": "2027-12-31" },
    { "project_id": "PRJ04_G03", "title": "Coorte de Doença Renal Crônica", "researcher_username": "pes.mendes", "condition_code": "CKD", "status": "PENDING", "valid_until": "2027-06-30" },
    { "project_id": "PRJ05_G03", "title": "Coorte de Insuficiência Cardíaca", "researcher_username": "pes.araujo", "condition_code": "HEART_FAILURE", "status": "EXPIRED", "valid_until": "2024-01-01" }
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
| o `id` no ANONYMIZED não é `hash8778a4` | o pseudônimo depende do `PSEUDONYM_SALT`; com outro salt o hash muda (é normal) |
| Postman não conecta | serviço fora do ar ou porta errada (gRPC é 50053); TLS desligado |
| métodos não aparecem no Postman | a reflection precisa do serviço no ar; recarregue após conectar, ou importe o `.proto` |
