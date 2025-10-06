package metrics

const (
	prefix = "foundations."
)

func FormatMetricName(name string) string {
	return prefix + name
}
