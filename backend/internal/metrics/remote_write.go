package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/gogo/protobuf/proto"
	"github.com/kurama/auction-system/backend/internal/logger"
	"go.uber.org/zap"

	promremote "github.com/prometheus/prometheus/prompb"
)

type RemoteWriteConfig struct {
	URL        string // Grafana Cloud remote write URL
	Username   string // Instance ID
	Password   string // API key
	PushPeriod time.Duration
}

// StartRemoteWrite starts a background goroutine that pushes metrics to Grafana Cloud.
func StartRemoteWrite(ctx context.Context, cfg RemoteWriteConfig) {
	if cfg.URL == "" || cfg.Username == "" || cfg.Password == "" {
		logger.Info("remote write disabled: missing config")
		return
	}

	period := cfg.PushPeriod
	if period == 0 {
		period = 15 * time.Second
	}

	logger.Info("remote write started", zap.String("url", cfg.URL), zap.Duration("period", period))

	go func() {
		ticker := time.NewTicker(period)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := pushMetrics(cfg); err != nil {
					logger.Error("remote write push failed", zap.Error(err))
				}
			}
		}
	}()
}

func pushMetrics(cfg RemoteWriteConfig) error {
	// Gather all metrics
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return fmt.Errorf("gather: %w", err)
	}

	// Convert to TimeSeries
	now := time.Now().UnixMilli()
	var timeSeries []promremote.TimeSeries

	for _, mf := range mfs {
		for _, m := range mf.GetMetric() {
			labels := []promremote.Label{
				{Name: "__name__", Value: mf.GetName()},
			}
			for _, lp := range m.GetLabel() {
				labels = append(labels, promremote.Label{Name: lp.GetName(), Value: lp.GetValue()})
			}

			var value float64
			switch mf.GetType() {
			case dto.MetricType_COUNTER:
				value = m.GetCounter().GetValue()
			case dto.MetricType_GAUGE:
				value = m.GetGauge().GetValue()
			case dto.MetricType_HISTOGRAM:
				// Push sum and count for histograms
				h := m.GetHistogram()
				timeSeries = append(timeSeries,
					promremote.TimeSeries{
						Labels:  appendLabel(labels, "__name__", mf.GetName()+"_sum"),
						Samples: []promremote.Sample{{Value: h.GetSampleSum(), Timestamp: now}},
					},
					promremote.TimeSeries{
						Labels:  appendLabel(labels, "__name__", mf.GetName()+"_count"),
						Samples: []promremote.Sample{{Value: float64(h.GetSampleCount()), Timestamp: now}},
					},
				)
				continue
			case dto.MetricType_SUMMARY:
				s := m.GetSummary()
				timeSeries = append(timeSeries,
					promremote.TimeSeries{
						Labels:  appendLabel(labels, "__name__", mf.GetName()+"_sum"),
						Samples: []promremote.Sample{{Value: s.GetSampleSum(), Timestamp: now}},
					},
					promremote.TimeSeries{
						Labels:  appendLabel(labels, "__name__", mf.GetName()+"_count"),
						Samples: []promremote.Sample{{Value: float64(s.GetSampleCount()), Timestamp: now}},
					},
				)
				continue
			default:
				continue
			}

			timeSeries = append(timeSeries, promremote.TimeSeries{
				Labels:  labels,
				Samples: []promremote.Sample{{Value: value, Timestamp: now}},
			})
		}
	}

	if len(timeSeries) == 0 {
		return nil
	}

	// Encode as protobuf + snappy
	writeReq := &promremote.WriteRequest{Timeseries: timeSeries}
	data, err := proto.Marshal(writeReq)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	compressed := snappy.Encode(nil, data)

	// Send HTTP request
	req, err := http.NewRequest("POST", cfg.URL, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")
	req.SetBasicAuth(cfg.Username, cfg.Password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remote write %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func appendLabel(labels []promremote.Label, name, value string) []promremote.Label {
	out := make([]promremote.Label, len(labels))
	copy(out, labels)
	for i := range out {
		if out[i].Name == name {
			out[i].Value = value
			return out
		}
	}
	return append(out, promremote.Label{Name: name, Value: value})
}

// Ensure expfmt is importable (used indirectly)
var _ = expfmt.TypeTextPlain
