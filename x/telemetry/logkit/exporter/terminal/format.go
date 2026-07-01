package terminal

import (
	"path/filepath"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

const (
	timeLayout      = "06-01-02T15:04:05"
	maxAttrValueLen = 120
	attrsPerLine    = 5
	emptyField      = "-"
)

var systemAttrs = map[string]struct{}{
	string(semconv.CodeFilePathKey):        {},
	string(semconv.CodeLineNumberKey):      {},
	string(semconv.CodeFunctionNameKey):    {},
	string(semconv.CodeStacktraceKey):      {},
	string(semconv.ExceptionStacktraceKey): {},
	string(semconv.ExceptionTypeKey):       {},
}

func attrDisplayKey(key string) (string, bool) {
	switch key {
	case string(semconv.ExceptionMessageKey), "error":
		return "error", true
	default:
		return key, false
	}
}

type recordMeta struct {
	filePath string
	line     string
	attrs    []string
}

func formatRecord(record sdklog.Record, colors palette) string {
	meta := collectMeta(record)

	var b strings.Builder

	b.WriteString(formatHeader(record, meta, colors))
	b.WriteByte('\n')
	b.WriteString(colors.body(formatBody(record.Body())))
	b.WriteByte('\n')

	if len(meta.attrs) > 0 {
		b.WriteString(formatAttrLines(meta.attrs, colors))
	}

	return b.String()
}

func formatHeader(record sdklog.Record, meta recordMeta, colors palette) string {
	ts := record.Timestamp()
	if ts.IsZero() {
		ts = record.ObservedTimestamp()
	}

	timeVal := emptyField
	if !ts.IsZero() {
		timeVal = ts.Format(timeLayout)
	}

	level := record.SeverityText()
	if level == "" {
		level = record.Severity().String()
	}

	if level == "" || level == log.SeverityUndefined.String() {
		level = emptyField
	}

	traceID := emptyField
	if id := record.TraceID(); id.IsValid() {
		traceID = id.String()
	}

	filePath := meta.filePath
	if filePath == "" {
		filePath = emptyField
	}

	line := meta.line
	if line == "" {
		line = emptyField
	}

	parts := []string{
		colors.time(timeVal),
		colors.level(level),
		colors.location(filePath),
		colors.location(line),
		colors.traceID(traceID),
	}

	return strings.Join(parts, " - ")
}

func collectMeta(record sdklog.Record) recordMeta {
	meta := recordMeta{
		attrs: make([]string, 0, record.AttributesLen()),
	}

	record.WalkAttributes(func(kv log.KeyValue) bool {
		switch kv.Key {
		case string(semconv.CodeFilePathKey):
			meta.filePath = shortenPath(kv.Value.AsString())

			return true
		case string(semconv.CodeLineNumberKey):
			meta.line = formatValue(kv.Value)

			return true
		}

		if _, skip := systemAttrs[kv.Key]; skip {
			return true
		}

		displayKey, _ := attrDisplayKey(kv.Key)
		meta.attrs = append(meta.attrs, formatAttrPair(displayKey, formatValue(kv.Value)))

		return true
	})

	return meta
}

func shortenPath(path string) string {
	if path == "" {
		return ""
	}

	short := filepath.Base(path)
	if short == "." || short == "/" {
		return path
	}

	return short
}

func formatBody(body log.Value) string {
	switch body.Kind() {
	case log.KindString:
		return body.AsString()
	case log.KindEmpty:
		return ""
	default:
		return truncateValue(formatValue(body))
	}
}

func formatAttrLines(attrs []string, colors palette) string {
	var b strings.Builder

	for i := 0; i < len(attrs); i += attrsPerLine {
		end := min(i+attrsPerLine, len(attrs))
		chunk := attrs[i:end]

		for j, attr := range chunk {
			if j > 0 {
				b.WriteByte('\t')
			}

			b.WriteString(colors.attr(attr))
		}

		b.WriteByte('\n')
	}

	return strings.TrimSuffix(b.String(), "\n")
}

func formatAttrPair(key, value string) string {
	return "<" + key + "=" + quoteValue(value) + ">"
}

func formatValue(v log.Value) string {
	switch v.Kind() {
	case log.KindString:
		return v.AsString()
	case log.KindInt64:
		return strconv.FormatInt(v.AsInt64(), 10)
	case log.KindFloat64:
		return strconv.FormatFloat(v.AsFloat64(), 'f', -1, 64)
	case log.KindBool:
		return strconv.FormatBool(v.AsBool())
	case log.KindBytes:
		return string(v.AsBytes())
	case log.KindSlice:
		items := v.AsSlice()

		parts := make([]string, 0, len(items))
		for _, item := range items {
			parts = append(parts, formatValue(item))
		}

		return "[" + strings.Join(parts, ",") + "]"
	case log.KindMap:
		pairs := v.AsMap()

		parts := make([]string, 0, len(pairs))
		for _, kv := range pairs {
			parts = append(parts, kv.Key+"="+formatValue(kv.Value))
		}

		return "{" + strings.Join(parts, " ") + "}"
	case log.KindEmpty:
		return ""
	default:
		return v.AsString()
	}
}

func quoteValue(val string) string {
	val = truncateValue(val)
	if val == "" {
		return `""`
	}

	if strings.ContainsAny(val, " \t\n\r\"<>") {
		return strconv.Quote(val)
	}

	return val
}

func truncateValue(val string) string {
	if len(val) <= maxAttrValueLen {
		return val
	}

	return val[:maxAttrValueLen] + "..."
}
