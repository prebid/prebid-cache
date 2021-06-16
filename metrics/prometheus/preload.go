package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

func preloadLabelValues(m *PrometheusMetrics) {
	preloadLabelValuesForCounter(m.Puts.RequestStatus, map[string][]string{StatusKey: {ErrorVal, BadRequestVal, TotalsVal, CustomKey}})
	preloadLabelValuesForCounter(m.Gets.RequestStatus, map[string][]string{StatusKey: {ErrorVal, BadRequestVal, TotalsVal}})
	preloadLabelValuesForCounter(m.PutsBackend.PutBackendRequests, map[string][]string{FormatKey: {XmlVal, JsonVal, InvFormatVal, DefinesTTLVal, ErrorVal}})
	preloadLabelValuesForCounter(m.GetsBackend.RequestStatus, map[string][]string{StatusKey: {ErrorVal, BadRequestVal, TotalsVal}})
	preloadLabelValuesForCounter(m.GetsBackend.ErrorsByType, map[string][]string{TypeKey: {KeyNotFoundVal, MissingKeyVal}})
	preloadLabelValuesForCounter(m.Connections.ConnectionsErrors, map[string][]string{ConnErrorKey: {CloseVal, AcceptVal}})
}

func preloadLabelValuesForCounter(counter *prometheus.CounterVec, labelsWithValues map[string][]string) {
	registerLabelPermutations(labelsWithValues, func(labels prometheus.Labels) {
		counter.With(labels)
	})
}

func preloadLabelValuesForHistogram(histogram *prometheus.HistogramVec, labelsWithValues map[string][]string) {
	registerLabelPermutations(labelsWithValues, func(labels prometheus.Labels) {
		histogram.With(labels)
	})
}

func registerLabelPermutations(labelsWithValues map[string][]string, register func(prometheus.Labels)) {
	if len(labelsWithValues) == 0 {
		return
	}

	keys := make([]string, 0, len(labelsWithValues))
	values := make([][]string, 0, len(labelsWithValues))
	for k, v := range labelsWithValues {
		keys = append(keys, k)
		values = append(values, v)
	}

	labels := prometheus.Labels{}
	registerLabelPermutationsRecursive(0, keys, values, labels, register)
}

func registerLabelPermutationsRecursive(depth int, keys []string, values [][]string, labels prometheus.Labels, register func(prometheus.Labels)) {
	label := keys[depth]
	isLeaf := depth == len(keys)-1

	if isLeaf {
		for _, v := range values[depth] {
			labels[label] = v
			register(cloneLabels(labels))
		}
	} else {
		for _, v := range values[depth] {
			labels[label] = v
			registerLabelPermutationsRecursive(depth+1, keys, values, labels, register)
		}
	}
}

func cloneLabels(labels prometheus.Labels) prometheus.Labels {
	clone := prometheus.Labels{}
	for k, v := range labels {
		clone[k] = v
	}
	return clone
}
