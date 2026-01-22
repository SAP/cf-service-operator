/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and cf-service-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	metricsNamespace = ""
	metricsSubsystem = "http_client"
)

// IndependentExecutionGeneral runs several operations independently of each other and only propagates the error value
func IndependentExecutionGeneral(fns ...func() error) error {
	var combinedError error
	for _, f := range fns {
		err := f()
		if err != nil {
			if combinedError == nil {
				combinedError = err
			} else {
				// TODO improve error wrapping
				// nolint:errorlint // we will improve the error wrapping in the future
				combinedError = fmt.Errorf("%s; %s", combinedError.Error(), err.Error())
			}
		}
	}

	return combinedError
}

// AddMetricsToTransport injects the Prometheus metrics to the HTTP transport
func AddMetricsToTransport(transport http.RoundTripper, registry prometheus.Registerer, target string, host string) (http.RoundTripper, error) {
	if transport == nil {
		transport = http.DefaultTransport
	}
	constLabels := prometheus.Labels{
		"target": target,
		"host":   host,
	}

	// create metrics
	inFlightGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "requests_in_flight",
		Subsystem:   metricsSubsystem,
		Namespace:   metricsNamespace,
		Help:        "The number in-flight requests for corresponding HTTP client",
		ConstLabels: constLabels,
	})
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "requests_total",
			Subsystem:   metricsSubsystem,
			Namespace:   metricsNamespace,
			Help:        "The number of HTTP requests from corresponding HTTP client",
			ConstLabels: constLabels,
		},
		[]string{"code", "method"},
	)
	histVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "request_duration_seconds",
			Subsystem:   metricsSubsystem,
			Namespace:   metricsNamespace,
			Help:        "A histogram of HTTP request latencies.",
			Buckets:     prometheus.DefBuckets,
			ConstLabels: constLabels,
		},
		[]string{},
	)

	// register created metrics or replace them by already registered ones
	err := IndependentExecutionGeneral(
		func() error {
			err := registry.Register(inFlightGauge)
			if err == nil {
				return nil
			}
			var e prometheus.AlreadyRegisteredError
			if errors.As(err, &e) {
				if collector, ok := e.ExistingCollector.(prometheus.Gauge); ok {
					inFlightGauge = collector
					return nil
				}
			}
			return err
		},
		func() error {
			err := registry.Register(counter)
			if err == nil {
				return nil
			}
			var e prometheus.AlreadyRegisteredError
			if errors.As(err, &e) {
				if collector, ok := e.ExistingCollector.(*prometheus.CounterVec); ok {
					counter = collector
					return nil
				}
			}
			return err
		},
		func() error {
			err := registry.Register(histVec)
			if err == nil {
				return nil
			}
			var e prometheus.AlreadyRegisteredError
			if errors.As(err, &e) {
				if collector, ok := e.ExistingCollector.(*prometheus.HistogramVec); ok {
					histVec = collector
					return nil
				}
			}
			return err
		},
	)

	transport = promhttp.InstrumentRoundTripperInFlight(inFlightGauge, transport)
	transport = promhttp.InstrumentRoundTripperCounter(counter, transport)
	transport = promhttp.InstrumentRoundTripperDuration(histVec, transport)

	return transport, err
}
