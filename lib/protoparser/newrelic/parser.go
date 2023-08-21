package newrelic

import (
	"fmt"
	"sync"
	"unicode"

	"github.com/valyala/fastjson"
	"github.com/valyala/fastjson/fastfloat"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/bytesutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
)

var baseEventKeys = map[string]struct{}{
	"timestamp": {}, "eventType": {},
}

var tagsPool = sync.Pool{
	New: func() interface{} {
		return make([]Tag, 0)
	},
}

// NewRelic agent sends next struct to the collector
// MetricPost entity item for the HTTP post to be sent to the ingest service.
// type MetricPost struct {
// 	ExternalKeys []string          `json:"ExternalKeys,omitempty"`
// 	EntityID     uint64            `json:"EntityID,omitempty"`
// 	IsAgent      bool              `json:"IsAgent"`
// 	Events       []json.RawMessage `json:"Events"`
// 	// Entity ID of the reporting agent, which will = EntityID when IsAgent == true.
// 	// The field is required in the backend for host metadata matching of the remote entities
// 	ReportingAgentID uint64 `json:"ReportingAgentID,omitempty"`
// }
// We are using only Events field because it contains all needed metrics

// Events represents Metrics collected from NewRelic MetricPost request
type Events struct {
	Metrics []Metric
}

// Unmarshal takes fastjson.Value and collects Metrics
func (e *Events) Unmarshal(v []*fastjson.Value) error {
	for _, value := range v {
		events := value.Get("Events")
		if events == nil {
			return fmt.Errorf("got empty Events array from request")
		}
		eventsArr, err := events.Array()
		if err != nil {
			return fmt.Errorf("error collect events: %s", err)
		}

		for _, event := range eventsArr {
			metricData, err := event.Object()
			if err != nil {
				return fmt.Errorf("error get metric data: %s", err)
			}
			var m Metric
			metrics, err := m.unmarshal(metricData)
			if err != nil {
				return fmt.Errorf("error collect metrics from Newrelic json: %s", err)
			}
			e.Metrics = append(e.Metrics, metrics...)
		}
	}

	return nil
}

// Metric represents VictoriaMetrics metrics
type Metric struct {
	Timestamp int64
	Tags      []Tag
	Metric    string
	Value     float64
}

func (m *Metric) unmarshal(o *fastjson.Object) ([]Metric, error) {
	m.reset()

	tags := tagsPool.Get().([]Tag)
	defer func() {
		tags = tags[:0]
		tagsPool.Put(tags)
	}()

	metrics := make([]Metric, 0, o.Len())
	rawTs := o.Get("timestamp")
	if rawTs != nil {
		ts, err := getFloat64(rawTs)
		if err != nil {
			return nil, fmt.Errorf("invalid `timestamp` in %s: %w", o, err)
		}
		m.Timestamp = int64(ts * 1e3)
	} else {
		// Allow missing timestamp. It is automatically populated
		// with the current time in this case.
		m.Timestamp = 0
	}

	eventType := o.Get("eventType")
	if eventType == nil {
		return nil, fmt.Errorf("error get eventType from Events object: %s", o)
	}

	o.Visit(func(key []byte, v *fastjson.Value) {

		k := bytesutil.ToUnsafeString(key)
		// skip already which has been parsed before
		// this is keys of BaseEvent type.
		// this type contains all NewRelic structs
		if _, ok := baseEventKeys[k]; ok {
			return
		}

		switch v.Type() {
		case fastjson.TypeString:
			// this is label with value
			name := k
			value := v.Get()
			if value == nil {
				logger.Errorf("error get NewRelic label value from json: %s", v)
				return
			}
			val := bytesutil.ToUnsafeString(value.GetStringBytes())
			tags = append(tags, Tag{Key: name, Value: val})
		case fastjson.TypeNumber:
			// this is metric name with value
			val := bytesutil.ToUnsafeString(eventType.GetStringBytes())
			mn := fmt.Sprintf("%s_%s", val, k)
			metricName := camelToSnakeCase(mn)
			f, err := getFloat64(v)
			if err != nil {
				logger.Errorf("error get NewRelic value for metric: %q; %s", k, err)
				return
			}
			metrics = append(metrics, Metric{Metric: metricName, Value: f})
		default:
			// unknown type
			logger.Errorf("got unsupported NewRelic json %s field type: %s", v, v.Type())
			return
		}
	})

	for i := range metrics {
		metrics[i].Timestamp = m.Timestamp
		metrics[i].Tags = tags
	}

	return metrics, nil
}

func (m *Metric) reset() {
	m.Timestamp = 0
	m.Tags = nil
	m.Metric = ""
	m.Value = 0
}

// Tag is an NewRelic tag.
type Tag struct {
	Key   string
	Value string
}

func camelToSnakeCase(str string) string {
	length := len(str)
	snakeCase := make([]byte, 0, length)

	for i := 0; i < length; i++ {
		char := str[i]
		if i > 0 && unicode.IsUpper(rune(char)) {
			snakeCase = append(snakeCase, '_')
		}
		snakeCase = append(snakeCase, byte(unicode.ToLower(rune(char))))
	}
	s := bytesutil.ToUnsafeString(snakeCase)
	return s
}

func getFloat64(v *fastjson.Value) (float64, error) {
	switch v.Type() {
	case fastjson.TypeNumber:
		return v.Float64()
	case fastjson.TypeString:
		vStr, _ := v.StringBytes()
		vFloat, err := fastfloat.Parse(bytesutil.ToUnsafeString(vStr))
		if err != nil {
			return 0, fmt.Errorf("cannot parse value %q: %w", vStr, err)
		}
		return vFloat, nil
	default:
		return 0, fmt.Errorf("value doesn't contain float64; it contains %s", v.Type())
	}
}
