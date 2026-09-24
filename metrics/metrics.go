package metrics

var Prefix = "foundations"

func FormatMetricName(name string) string { return Prefix + "." + name }
