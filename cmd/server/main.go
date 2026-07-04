// Command server sobe o data-transform-service: servidor gRPC + servidor HTTP de métricas.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	dtpb "github.com/pspd-2026-2-trabalho-2/data-transform-service/gen/datatransform/v1"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/anonymize"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/config"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/grpcserver"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/observability"
	"github.com/pspd-2026-2-trabalho-2/data-transform-service/internal/service"
)

const grpcServiceName = "datatransform.v1.DataTransformService"

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	metrics := observability.NewMetrics()
	anon := anonymize.New(cfg.PseudonymSalt)
	svc := service.New(anon, metrics)
	srv := grpcserver.New(svc)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			metrics.UnaryServerInterceptor(),
			loggingInterceptor(logger),
		),
	)
	dtpb.RegisterDataTransformServiceServer(grpcServer, srv)

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus(grpcServiceName, healthpb.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	metricsSrv := startMetricsServer(cfg.MetricsPort, logger)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logger.Error("falha ao abrir porta gRPC", "port", cfg.GRPCPort, "err", err)
		os.Exit(1)
	}
	go func() {
		logger.Info("servidor gRPC ouvindo", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("servidor gRPC parou", "err", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("encerrando (graceful shutdown)...")
	healthSrv.Shutdown()
	grpcServer.GracefulStop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = metricsSrv.Shutdown(shutdownCtx)

	logger.Info("finalizado")
}

func startMetricsServer(port string, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Addr: ":" + port, Handler: mux}
	go func() {
		logger.Info("servidor de métricas ouvindo", "port", port, "path", "/metrics")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("servidor de métricas parou", "err", err)
		}
	}()
	return srv
}

func loggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		logger.Info("rpc",
			"method", info.FullMethod,
			"duration_ms", time.Since(start).Milliseconds(),
			"err", err,
		)
		return resp, err
	}
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
