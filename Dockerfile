# ---- build ----
FROM golang:1.25 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/data-transform-service ./cmd/server

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/data-transform-service /data-transform-service
EXPOSE 50053 9091
USER nonroot:nonroot
ENTRYPOINT ["/data-transform-service"]
