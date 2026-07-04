// Package observability concentra as métricas Prometheus do serviço.
package observability

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type Metrics struct {
	rpcHandled  *prometheus.CounterVec
	rpcDuration *prometheus.HistogramVec

	transformsTotal *prometheus.CounterVec
}

// NewMetrics registra os coletores no registrador default (chamar uma vez).
func NewMetrics() *Metrics {
	return &Metrics{
		rpcHandled: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_server_handled_total",
			Help: "Total de RPCs finalizadas, por método e código de status.",
		}, []string{"method", "code"}),
		rpcDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "grpc_server_handling_seconds",
			Help:    "Latência das RPCs em segundos, por método.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method"}),
		transformsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "fhir_transforms_total",
			Help: "Total de transformações FHIR realizadas, por operação e nível de acesso.",
		}, []string{"operation", "level"}),
	}
}

func (m *Metrics) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		m.rpcHandled.WithLabelValues(info.FullMethod, status.Code(err).String()).Inc()
		m.rpcDuration.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())
		return resp, err
	}
}

func (m *Metrics) RecordTransform(operation, level string) {
	m.transformsTotal.WithLabelValues(operation, level).Inc()
}
