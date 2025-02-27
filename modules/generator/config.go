package generator

import (
	"flag"
	"time"

	"github.com/pkg/errors"

	"github.com/grafana/tempo/modules/generator/processor/localblocks"
	"github.com/grafana/tempo/modules/generator/processor/servicegraphs"
	"github.com/grafana/tempo/modules/generator/processor/spanmetrics"
	"github.com/grafana/tempo/modules/generator/registry"
	"github.com/grafana/tempo/modules/generator/storage"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
)

const (
	// RingKey is the key under which we store the metric-generator's ring in the KVStore.
	RingKey = "metrics-generator"

	// ringNameForServer is the name of the ring used by the metrics-generator server.
	ringNameForServer = "metrics-generator"
)

// Config for a generator.
type Config struct {
	Ring      RingConfig      `yaml:"ring"`
	Processor ProcessorConfig `yaml:"processor"`
	Registry  registry.Config `yaml:"registry"`
	Storage   storage.Config  `yaml:"storage"`
	TracesWAL wal.Config      `yaml:"traces_storage"`
	// MetricsIngestionSlack is the max amount of time passed since a span's start time
	// for the span to be considered in metrics generation
	MetricsIngestionSlack time.Duration `yaml:"metrics_ingestion_time_range_slack"`
}

// RegisterFlagsAndApplyDefaults registers the flags.
func (cfg *Config) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.Ring.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Processor.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Registry.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.Storage.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.TracesWAL.Version = encoding.DefaultEncoding().Version()

	// setting default for max span age before discarding to 30s
	cfg.MetricsIngestionSlack = 30 * time.Second
}

type ProcessorConfig struct {
	ServiceGraphs servicegraphs.Config `yaml:"service_graphs"`
	SpanMetrics   spanmetrics.Config   `yaml:"span_metrics"`
	LocalBlocks   localblocks.Config   `yaml:"local_blocks"`
}

func (cfg *ProcessorConfig) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	cfg.ServiceGraphs.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.SpanMetrics.RegisterFlagsAndApplyDefaults(prefix, f)
	cfg.LocalBlocks.RegisterFlagsAndApplyDefaults(prefix, f)
}

// copyWithOverrides creates a copy of the config using values set in the overrides.
func (cfg *ProcessorConfig) copyWithOverrides(o metricsGeneratorOverrides, userID string) (ProcessorConfig, error) {
	copyCfg := *cfg

	if buckets := o.MetricsGeneratorProcessorServiceGraphsHistogramBuckets(userID); buckets != nil {
		copyCfg.ServiceGraphs.HistogramBuckets = buckets
	}
	if dimensions := o.MetricsGeneratorProcessorServiceGraphsDimensions(userID); dimensions != nil {
		copyCfg.ServiceGraphs.Dimensions = dimensions
	}
	if buckets := o.MetricsGeneratorProcessorSpanMetricsHistogramBuckets(userID); buckets != nil {
		copyCfg.SpanMetrics.HistogramBuckets = buckets
	}
	if dimensions := o.MetricsGeneratorProcessorSpanMetricsDimensions(userID); dimensions != nil {
		copyCfg.SpanMetrics.Dimensions = dimensions
	}
	if dimensions := o.MetricsGeneratorProcessorSpanMetricsIntrinsicDimensions(userID); dimensions != nil {
		err := copyCfg.SpanMetrics.IntrinsicDimensions.ApplyFromMap(dimensions)
		if err != nil {
			return ProcessorConfig{}, errors.Wrap(err, "fail to apply overrides")
		}
	}
	if filterPolicies := o.MetricsGeneratorProcessorSpanMetricsFilterPolicies(userID); filterPolicies != nil {
		copyCfg.SpanMetrics.FilterPolicies = filterPolicies
	}

	if max := o.MetricsGeneratorProcessorLocalBlocksMaxLiveTraces(userID); max > 0 {
		copyCfg.LocalBlocks.MaxLiveTraces = max
	}

	if max := o.MetricsGeneratorProcessorLocalBlocksMaxBlockDuration(userID); max > 0 {
		copyCfg.LocalBlocks.MaxBlockDuration = max
	}

	if max := o.MetricsGeneratorProcessorLocalBlocksMaxBlockBytes(userID); max > 0 {
		copyCfg.LocalBlocks.MaxBlockBytes = max
	}

	if period := o.MetricsGeneratorProcessorLocalBlocksFlushCheckPeriod(userID); period > 0 {
		copyCfg.LocalBlocks.FlushCheckPeriod = period
	}

	if period := o.MetricsGeneratorProcessorLocalBlocksTraceIdlePeriod(userID); period > 0 {
		copyCfg.LocalBlocks.TraceIdlePeriod = period
	}

	if timeout := o.MetricsGeneratorProcessorLocalBlocksCompleteBlockTimeout(userID); timeout > 0 {
		copyCfg.LocalBlocks.CompleteBlockTimeout = timeout
	}

	return copyCfg, nil
}
