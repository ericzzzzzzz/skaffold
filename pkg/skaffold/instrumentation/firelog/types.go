package firelog

import (
	"bytes"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type ClientInfo struct {
	ClientType string `json:"client_type"`
}

type MetricData struct {
	ClientInfo      ClientInfo `json:"client_info"`
	LogSource       string     `json:"log_source"`
	LogEvent        LogEvent   `json:"log_event"`
	RequestTimeMS   int64      `json:"request_time_ms"`
	RequestUptimeMS int64      `json:"request_uptime_ms"`
}

type LogEvent struct {
	EventTimeMS                  int64  `json:"event_time_ms"`
	EventUptimeMS                int64  `json:"event_uptime_ms"`
	SourceExtensionJsonProto3Str string `json:"source_extension_json_proto3"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type EventMetadata []KeyValue

type SourceExtensionJsonProto3 struct {
	ProjectID       string     `json:"project_id"`
	ConsoleType     string     `json:"console_type"`
	ClientInstallID string     `json:"client_install_id"`
	EventName       string     `json:"event_name"`
	EventMetadata   []KeyValue `json:"event_metadata"`
}

type Key string

func ofKey(s string) Key {
	return Key(s)
}

func (k Key) Value(v string) KeyValue {
	return KeyValue{
		Key:   string(k),
		Value: v,
	}
}

func (md MetricData) newReader() *bytes.Reader {
	data, _ := json.Marshal(md)
	return bytes.NewReader(data)
}

type DataPoint interface {
	DataPointInt64 | DataPointFloat64 | DataPointHistogram
	value() string
	attributes() attribute.Set
	eventTime() int64
	upTime() int64
}

type DataPointInt64 metricdata.DataPoint[int64]

func (d DataPointInt64) attributes() attribute.Set {
	return d.Attributes
}

func (d DataPointInt64) eventTime() int64 {
	return d.StartTime.UnixMilli()
}

func (d DataPointInt64) upTime() int64 {
	return d.Time.UnixMilli()
}

type DataPointFloat64 metricdata.DataPoint[float64]

func (d DataPointFloat64) attributes() attribute.Set {
	return d.Attributes
}

func (d DataPointFloat64) eventTime() int64 {
	return d.StartTime.UnixMilli()
}

func (d DataPointFloat64) upTime() int64 {
	return d.Time.UnixMilli()
}

type DataPointHistogram metricdata.HistogramDataPoint

func (d DataPointHistogram) attributes() attribute.Set {
	return d.Attributes
}

func (d DataPointHistogram) eventTime() int64 {
	return d.StartTime.UnixMilli()
}

func (d DataPointHistogram) upTime() int64 {
	return d.Time.UnixMilli()
}

func (d DataPointInt64) value() string {
	return fmt.Sprintf("%d", d.Value)
}

func (d DataPointFloat64) value() string {
	return fmt.Sprintf("%f", d.Value)
}

func (d DataPointHistogram) value() string {
	return fmt.Sprintf("%f", d.Sum)
}
