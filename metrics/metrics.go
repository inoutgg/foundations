package metrics

var Prefix = "foundations" //nolint:gochecknoglobals

func FormatMetricName(name string) string { return Prefix + "." + name }
