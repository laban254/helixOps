package tempo

import "fmt"

// BuildServiceQuery constructs a TraceQL query to find all traces involving a specific service.
func BuildServiceQuery(serviceName string) string {
	// TraceQL syntax for finding any span with resource.service.name matching serviceName
	return fmt.Sprintf("{ resource.service.name = \"%s\" }", serviceName)
}

// BuildSlowSpansQuery constructs a TraceQL query to discover spans for a service that exceed a given latency threshold.
func BuildSlowSpansQuery(serviceName string, thresholdMs int) string {
	return fmt.Sprintf("{ resource.service.name = \"%s\" && duration > %dms }", serviceName, thresholdMs)
}

// BuildErrorSpansQuery constructs a TraceQL query to retrieve spans marked with an error status for a specific service.
func BuildErrorSpansQuery(serviceName string) string {
	return fmt.Sprintf("{ resource.service.name = \"%s\" && status = \"error\" }", serviceName)
}
