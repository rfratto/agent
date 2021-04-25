package config

import "regexp"

// trimMap recursively deletes fields from m whose value is nil.
func trimMap(m map[string]interface{}) {
	for k, v := range m {
		if v == nil {
			delete(m, k)
			continue
		}

		if next, ok := v.(map[string]interface{}); ok {
			trimMap(next)
		}

		if arr, ok := v.([]interface{}); ok {
			m[k] = trimSlice(arr)
		}
	}
}

func trimSlice(s []interface{}) []interface{} {
	res := make([]interface{}, 0, len(s))

	for _, e := range s {
		if e == nil {
			continue
		}

		if next, ok := e.([]interface{}); ok {
			e = trimSlice(next)
		}

		if next, ok := e.(map[string]interface{}); ok {
			trimMap(next)
		}

		res = append(res, e)
	}

	return res
}

var invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// SanitizeLabelName sanitizes a label name for Prometheus.
func SanitizeLabelName(name string) string {
	return invalidLabelCharRE.ReplaceAllString(name, "_")
}
