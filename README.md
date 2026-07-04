# data-transform-service

Microsserviço **gRPC em Go** do projeto PSPD (Trabalho 2). É a **camada de transformação**
do backend: recebe **dados clínicos crus** + o **nível de acesso**, aplica as regras de acesso
(anonimização / pseudonimização / agregação) e converte para **HL7/FHIR** antes de a resposta
chegar ao usuário.

É um **transformador puro**: **não acessa banco** e **não chama outros serviços** — os dados clínicos
crus chegam prontos no próprio request. Quem orquestra é a API Gateway: ela busca os dados no
[patient-data-service](https://github.com/pspd-2026-2-trabalho-2/patient-data-service) e os envia
para cá junto com o nível de acesso.

```
                      ┌─▶ patient-data-service ─▶ PostgreSQL      (dados crus)
API Gateway ──────────┤
 (valida JWT,         └─▶ data-transform-service                 (dados crus + nível → FHIR)
  orquestra e consolida)
```

## Níveis de acesso

| Campo | FULL | PARTIAL | ANONYMIZED |
|---|---|---|---|
| id | real | real | pseudônimo `hash…` (SHA-256 com salt) |
| nome | completo | iniciais (`J.S.`) | removido |
| CPF / CNS | incluídos | removidos | removidos |
| nascimento | data exata | só o ano | removida (vira faixa etária) |
| sexo | sim | sim | sim |
| cidade | sim | sim | removida |
| estado | sim | sim | sim |

**AGGREGATED** não retorna paciente algum — só totais e percentuais (`TransformCohortStatistics`).

## Mapeamento HL7/FHIR

| Origem (patient-data) | Recurso FHIR |
|---|---|
| patients | `Patient` |
| encounters | `Encounter` |
| clinical_events (Condition) | `Condition` |
| clinical_events (Observation) | `Observation` |
| clinical_events (Medication) | `MedicationRequest` |
| projects | `ResearchStudy` |

Vários recursos são agrupados num `Bundle`. A saída é o próprio objeto FHIR (recurso ou Bundle),
como `google.protobuf.Struct` — ou seja, JSON, não uma string escapada.

## Requisitos
- Go 1.25+, Docker + Compose
- `protoc` + plugins Go (só para regerar o código; já vem gerado em `gen/`)
- Opcional: `grpcurl`

Não precisa de banco nem do patient-data rodando: os dados chegam no request.

## Estrutura
```
proto/datatransform/v1/    contrato próprio (server); importa os tipos de dados abaixo
proto/patientdata/v1/      cópia das MENSAGENS do patient-data (sem service; usadas nos requests)
gen/                       código gerado
cmd/server/main.go         entrypoint
internal/config            variáveis de ambiente
internal/anonymize         regras por nível + pseudonimização (puro, testável)
internal/fhir              structs FHIR + Bundle + JSON
internal/service           transformação (anonimiza → FHIR)
internal/grpcserver        transporte gRPC
internal/observability     métricas Prometheus
```

## Como executar

```bash
# via Docker
docker compose up -d --build

# ou via Go
cp .env.example .env
go run ./cmd/server
```

O serviço sobe em: gRPC `localhost:50053`, métricas/health HTTP `localhost:9091`.

## Variáveis de ambiente

| Variável | Padrão | Descrição |
|---|---|---|
| `GRPC_PORT` | `50053` | porta gRPC |
| `METRICS_PORT` | `9091` | porta HTTP de métricas/health |
| `PSEUDONYM_SALT` | `pspd-troque-este-salt` | salt do hash de pseudonimização |
| `LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |

## RPCs

Todos recebem os **dados crus** no request (tipos `patientdata.v1.*`) e, quando aplicável, o `access_level`.

| RPC | Entrada | Saída |
|---|---|---|
| `TransformPatient` | `patient`, `access_level` | FHIR Patient |
| `TransformClinicalSummary` | `summary`, `access_level` | Bundle (resumo clínico) |
| `TransformClinicalHistory` | `patient`, `events[]`, `access_level` | Bundle (histórico) |
| `TransformPatientList` | `patients[]`, `access_level` | Bundle de Patient |
| `TransformCohortExams` | `patients[]` (paciente+exames), `access_level` | Bundle anonimizado |
| `TransformCohortStatistics` | `stats` (contagens cruas) | estatísticas com percentuais |
| `TransformProjects` | `projects[]` | Bundle de ResearchStudy |

`AccessLevel`: `FULL`, `PARTIAL`, `ANONYMIZED`, `AGGREGATED`.

## Testes

Guia completo (como subir, casos de teste, payloads e resultados esperados) em
**[TESTES.md](TESTES.md)**. Resumo rápido:

```bash
go test ./...   # unitários: regras de anonimização e conversão FHIR
```

## Observabilidade
`GET http://localhost:9091/metrics` expõe:
- `grpc_server_handled_total{method,code}` / `grpc_server_handling_seconds` — RPCs
- `fhir_transforms_total{operation,level}` — transformações por operação e nível
- `go_*` / `process_*` — CPU, memória, goroutines

## Testes
```bash
go test ./...   # unit: regras de anonimização e conversão FHIR
```

## Regenerar o código do proto
```bash
make tools   # instala protoc-gen-go e protoc-gen-go-grpc
make proto   # gera mensagens (datatransform + patientdata) e o service datatransform
```
Se o contrato de dados do patient-data mudar, atualize as mensagens em `proto/patientdata/v1/` e rode `make proto`.
