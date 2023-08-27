package rule

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/app/vmalert/config"
	"github.com/VictoriaMetrics/VictoriaMetrics/app/vmalert/datasource"
	"github.com/VictoriaMetrics/VictoriaMetrics/app/vmalert/notifier"
	"github.com/VictoriaMetrics/VictoriaMetrics/app/vmalert/templates"
	"github.com/VictoriaMetrics/VictoriaMetrics/app/vmalert/utils"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/prompbmarshal"
)

// AlertingRule is basic alert entity
type AlertingRule struct {
	Type          config.Type
	RuleID        uint64
	Name          string
	Expr          string
	For           time.Duration
	KeepFiringFor time.Duration
	Labels        map[string]string
	Annotations   map[string]string
	GroupID       uint64
	GroupName     string
	EvalInterval  time.Duration
	Debug         bool

	q datasource.Querier

	AlertsMu sync.RWMutex
	// stores list of active alerts
	Alerts map[uint64]*notifier.Alert

	// state stores recent state changes
	// during evaluations
	State *ruleState

	metrics *alertingRuleMetrics
}

type alertingRuleMetrics struct {
	errors        *utils.Gauge
	pending       *utils.Gauge
	active        *utils.Gauge
	samples       *utils.Gauge
	seriesFetched *utils.Gauge
}

func newAlertingRule(qb datasource.QuerierBuilder, group *Group, cfg config.Rule) *AlertingRule {
	ar := &AlertingRule{
		Type:          group.Type,
		RuleID:        cfg.ID,
		Name:          cfg.Alert,
		Expr:          cfg.Expr,
		For:           cfg.For.Duration(),
		KeepFiringFor: cfg.KeepFiringFor.Duration(),
		Labels:        cfg.Labels,
		Annotations:   cfg.Annotations,
		GroupID:       group.ID(),
		GroupName:     group.Name,
		EvalInterval:  group.Interval,
		Debug:         cfg.Debug,
		q: qb.BuildWithParams(datasource.QuerierParams{
			DataSourceType:     group.Type.String(),
			EvaluationInterval: group.Interval,
			QueryParams:        group.Params,
			Headers:            group.Headers,
			Debug:              cfg.Debug,
		}),
		Alerts:  make(map[uint64]*notifier.Alert),
		metrics: &alertingRuleMetrics{},
	}

	if cfg.UpdateEntriesLimit != nil {
		ar.State = NewRuleState(*cfg.UpdateEntriesLimit)
	} else {
		ar.State = NewRuleState(*ruleUpdateEntriesLimit)
	}

	labels := fmt.Sprintf(`alertname=%q, group=%q, id="%d"`, ar.Name, group.Name, ar.ID())
	ar.metrics.pending = utils.GetOrCreateGauge(fmt.Sprintf(`vmalert_alerts_pending{%s}`, labels),
		func() float64 {
			ar.AlertsMu.RLock()
			defer ar.AlertsMu.RUnlock()
			var num int
			for _, a := range ar.Alerts {
				if a.State == notifier.StatePending {
					num++
				}
			}
			return float64(num)
		})
	ar.metrics.active = utils.GetOrCreateGauge(fmt.Sprintf(`vmalert_alerts_firing{%s}`, labels),
		func() float64 {
			ar.AlertsMu.RLock()
			defer ar.AlertsMu.RUnlock()
			var num int
			for _, a := range ar.Alerts {
				if a.State == notifier.StateFiring {
					num++
				}
			}
			return float64(num)
		})
	ar.metrics.errors = utils.GetOrCreateGauge(fmt.Sprintf(`vmalert_alerting_rules_error{%s}`, labels),
		func() float64 {
			e := ar.State.GetLast()
			if e.Err == nil {
				return 0
			}
			return 1
		})
	ar.metrics.samples = utils.GetOrCreateGauge(fmt.Sprintf(`vmalert_alerting_rules_last_evaluation_samples{%s}`, labels),
		func() float64 {
			e := ar.State.GetLast()
			return float64(e.Samples)
		})
	ar.metrics.seriesFetched = utils.GetOrCreateGauge(fmt.Sprintf(`vmalert_alerting_rules_last_evaluation_series_fetched{%s}`, labels),
		func() float64 {
			e := ar.State.GetLast()
			if e.SeriesFetched == nil {
				// means seriesFetched is unsupported
				return -1
			}
			seriesFetched := float64(*e.SeriesFetched)
			if seriesFetched == 0 && e.Samples > 0 {
				// `alert: 0.95` will fetch no series
				// but will get one time series in response.
				seriesFetched = float64(e.Samples)
			}
			return seriesFetched
		})
	return ar
}

// close unregisters rule metrics
func (ar *AlertingRule) close() {
	ar.metrics.active.Unregister()
	ar.metrics.pending.Unregister()
	ar.metrics.errors.Unregister()
	ar.metrics.samples.Unregister()
	ar.metrics.seriesFetched.Unregister()
}

// String implements Stringer interface
func (ar *AlertingRule) String() string {
	return ar.Name
}

// ID returns unique Rule ID
// within the parent Group.
func (ar *AlertingRule) ID() uint64 {
	return ar.RuleID
}

func (ar *AlertingRule) logDebugf(at time.Time, a *notifier.Alert, format string, args ...interface{}) {
	if !ar.Debug {
		return
	}
	prefix := fmt.Sprintf("DEBUG rule %q:%q (%d) at %v: ",
		ar.GroupName, ar.Name, ar.RuleID, at.Format(time.RFC3339))

	if a != nil {
		labelKeys := make([]string, len(a.Labels))
		var i int
		for k := range a.Labels {
			labelKeys[i] = k
			i++
		}
		sort.Strings(labelKeys)
		labels := make([]string, len(labelKeys))
		for i, l := range labelKeys {
			labels[i] = fmt.Sprintf("%s=%q", l, a.Labels[l])
		}
		labelsStr := strings.Join(labels, ",")
		prefix += fmt.Sprintf("alert %d {%s} ", a.ID, labelsStr)
	}
	msg := fmt.Sprintf(format, args...)
	logger.Infof("%s", prefix+msg)
}

type labelSet struct {
	// origin labels extracted from received time series
	// plus extra labels (group labels, service labels like alertNameLabel).
	// in case of conflicts, origin labels from time series preferred.
	// used for templating annotations
	origin map[string]string
	// processed labels includes origin labels
	// plus extra labels (group labels, service labels like alertNameLabel).
	// in case of conflicts, extra labels are preferred.
	// used as labels attached to notifier.Alert and ALERTS series written to remote storage.
	processed map[string]string
}

// toLabels converts labels from given Metric
// to labelSet which contains original and processed labels.
func (ar *AlertingRule) toLabels(m datasource.Metric, qFn templates.QueryFn) (*labelSet, error) {
	ls := &labelSet{
		origin:    make(map[string]string),
		processed: make(map[string]string),
	}
	for _, l := range m.Labels {
		ls.origin[l.Name] = l.Value
		// drop __name__ to be consistent with Prometheus alerting
		if l.Name == "__name__" {
			continue
		}
		ls.processed[l.Name] = l.Value
	}

	extraLabels, err := notifier.ExecTemplate(qFn, ar.Labels, notifier.AlertTplData{
		Labels: ls.origin,
		Value:  m.Values[0],
		Expr:   ar.Expr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to expand labels: %s", err)
	}
	for k, v := range extraLabels {
		ls.processed[k] = v
		if _, ok := ls.origin[k]; !ok {
			ls.origin[k] = v
		}
	}

	// set additional labels to identify group and rule name
	if ar.Name != "" {
		ls.processed[alertNameLabel] = ar.Name
		if _, ok := ls.origin[alertNameLabel]; !ok {
			ls.origin[alertNameLabel] = ar.Name
		}
	}
	if !*disableAlertGroupLabel && ar.GroupName != "" {
		ls.processed[alertGroupNameLabel] = ar.GroupName
		if _, ok := ls.origin[alertGroupNameLabel]; !ok {
			ls.origin[alertGroupNameLabel] = ar.GroupName
		}
	}
	return ls, nil
}

// execRange executes alerting rule on the given time range similarly to exec.
// It doesn't update internal states of the Rule and meant to be used just
// to get time series for backfilling.
// It returns ALERT and ALERT_FOR_STATE time series as result.
func (ar *AlertingRule) execRange(ctx context.Context, start, end time.Time) ([]prompbmarshal.TimeSeries, error) {
	res, err := ar.q.QueryRange(ctx, ar.Expr, start, end)
	if err != nil {
		return nil, err
	}
	var result []prompbmarshal.TimeSeries
	qFn := func(query string) ([]datasource.Metric, error) {
		return nil, fmt.Errorf("`query` template isn't supported in replay mode")
	}
	for _, s := range res.Data {
		a, err := ar.newAlert(s, nil, time.Time{}, qFn) // initial alert
		if err != nil {
			return nil, fmt.Errorf("failed to create alert: %s", err)
		}
		if ar.For == 0 { // if alert is instant
			a.State = notifier.StateFiring
			for i := range s.Values {
				result = append(result, ar.alertToTimeSeries(a, s.Timestamps[i])...)
			}
			continue
		}

		// if alert with For > 0
		prevT := time.Time{}
		for i := range s.Values {
			at := time.Unix(s.Timestamps[i], 0)
			if at.Sub(prevT) > ar.EvalInterval {
				// reset to Pending if there are gaps > EvalInterval between DPs
				a.State = notifier.StatePending
				a.ActiveAt = at
			} else if at.Sub(a.ActiveAt) >= ar.For {
				a.State = notifier.StateFiring
				a.Start = at
			}
			prevT = at
			result = append(result, ar.alertToTimeSeries(a, s.Timestamps[i])...)
		}
	}
	return result, nil
}

// resolvedRetention is the duration for which a resolved alert instance
// is kept in memory state and consequently repeatedly sent to the AlertManager.
const resolvedRetention = 15 * time.Minute

// exec executes AlertingRule expression via the given Querier.
// Based on the Querier results AlertingRule maintains notifier.Alerts
func (ar *AlertingRule) exec(ctx context.Context, ts time.Time, limit int) ([]prompbmarshal.TimeSeries, error) {
	start := time.Now()
	res, req, err := ar.q.Query(ctx, ar.Expr, ts)
	curState := StateEntry{
		Time:          start,
		At:            ts,
		Duration:      time.Since(start),
		Samples:       len(res.Data),
		SeriesFetched: res.SeriesFetched,
		Err:           err,
		Curl:          requestToCurl(req),
	}

	defer func() {
		ar.State.Add(curState)
	}()

	ar.AlertsMu.Lock()
	defer ar.AlertsMu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to execute query %q: %w", ar.Expr, err)
	}

	ar.logDebugf(ts, nil, "query returned %d samples (elapsed: %s)", curState.Samples, curState.Duration)

	for h, a := range ar.Alerts {
		// cleanup inactive alerts from previous Exec
		if a.State == notifier.StateInactive && ts.Sub(a.ResolvedAt) > resolvedRetention {
			ar.logDebugf(ts, a, "deleted as inactive")
			delete(ar.Alerts, h)
		}
	}

	qFn := func(query string) ([]datasource.Metric, error) {
		res, _, err := ar.q.Query(ctx, query, ts)
		return res.Data, err
	}
	updated := make(map[uint64]struct{})
	// update list of active alerts
	for _, m := range res.Data {
		ls, err := ar.toLabels(m, qFn)
		if err != nil {
			curState.Err = fmt.Errorf("failed to expand labels: %s", err)
			return nil, curState.Err
		}
		h := hash(ls.processed)
		if _, ok := updated[h]; ok {
			// duplicate may be caused by extra labels
			// conflicting with the metric labels
			curState.Err = fmt.Errorf("labels %v: %w", ls.processed, errDuplicate)
			return nil, curState.Err
		}
		updated[h] = struct{}{}
		if a, ok := ar.Alerts[h]; ok {
			if a.State == notifier.StateInactive {
				// alert could be in inactive state for resolvedRetention
				// so when we again receive metrics for it - we switch it
				// back to notifier.StatePending
				a.State = notifier.StatePending
				a.ActiveAt = ts
				ar.logDebugf(ts, a, "INACTIVE => PENDING")
			}
			a.Value = m.Values[0]
			// re-exec template since Value or query can be used in annotations
			a.Annotations, err = a.ExecTemplate(qFn, ls.origin, ar.Annotations)
			if err != nil {
				return nil, err
			}
			a.KeepFiringSince = time.Time{}
			continue
		}
		a, err := ar.newAlert(m, ls, start, qFn)
		if err != nil {
			curState.Err = fmt.Errorf("failed to create alert: %w", err)
			return nil, curState.Err
		}
		a.ID = h
		a.State = notifier.StatePending
		a.ActiveAt = ts
		ar.Alerts[h] = a
		ar.logDebugf(ts, a, "created in state PENDING")
	}
	var numActivePending int
	for h, a := range ar.Alerts {
		// if alert wasn't updated in this iteration
		// means it is resolved already
		if _, ok := updated[h]; !ok {
			if a.State == notifier.StatePending {
				// alert was in Pending state - it is not
				// active anymore
				delete(ar.Alerts, h)
				ar.logDebugf(ts, a, "PENDING => DELETED: is absent in current evaluation round")
				continue
			}
			// check if alert should keep StateFiring if rule has
			// `keep_firing_for` field
			if a.State == notifier.StateFiring {
				if ar.KeepFiringFor > 0 {
					if a.KeepFiringSince.IsZero() {
						a.KeepFiringSince = ts
					}
				}
				// alerts with ar.KeepFiringFor>0 may remain FIRING
				// even if their expression isn't true anymore
				if ts.Sub(a.KeepFiringSince) > ar.KeepFiringFor {
					a.State = notifier.StateInactive
					a.ResolvedAt = ts
					ar.logDebugf(ts, a, "FIRING => INACTIVE: is absent in current evaluation round")
					continue
				}
				ar.logDebugf(ts, a, "KEEP_FIRING: will keep firing for %fs since %v", ar.KeepFiringFor.Seconds(), a.KeepFiringSince)
			}
		}
		numActivePending++
		if a.State == notifier.StatePending && ts.Sub(a.ActiveAt) >= ar.For {
			a.State = notifier.StateFiring
			a.Start = ts
			alertsFired.Inc()
			ar.logDebugf(ts, a, "PENDING => FIRING: %s since becoming active at %v", ts.Sub(a.ActiveAt), a.ActiveAt)
		}
	}
	if limit > 0 && numActivePending > limit {
		ar.Alerts = map[uint64]*notifier.Alert{}
		curState.Err = fmt.Errorf("exec exceeded limit of %d with %d alerts", limit, numActivePending)
		return nil, curState.Err
	}
	return ar.toTimeSeries(ts.Unix()), nil
}

func (ar *AlertingRule) toTimeSeries(timestamp int64) []prompbmarshal.TimeSeries {
	var tss []prompbmarshal.TimeSeries
	for _, a := range ar.Alerts {
		if a.State == notifier.StateInactive {
			continue
		}
		ts := ar.alertToTimeSeries(a, timestamp)
		tss = append(tss, ts...)
	}
	return tss
}

// updateWith copies all significant fields.
// alerts state isn't copied since
// it should be updated in next 2 Execs
func (ar *AlertingRule) updateWith(r Rule) error {
	nr, ok := r.(*AlertingRule)
	if !ok {
		return fmt.Errorf("BUG: attempt to update alerting rule with wrong type %#v", r)
	}
	ar.Expr = nr.Expr
	ar.For = nr.For
	ar.KeepFiringFor = nr.KeepFiringFor
	ar.Labels = nr.Labels
	ar.Annotations = nr.Annotations
	ar.EvalInterval = nr.EvalInterval
	ar.Debug = nr.Debug
	ar.q = nr.q
	ar.State = nr.State
	return nil
}

// TODO: consider hashing algorithm in VM
func hash(labels map[string]string) uint64 {
	hash := fnv.New64a()
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		// drop __name__ to be consistent with Prometheus alerting
		if k == "__name__" {
			continue
		}
		name, value := k, labels[k]
		hash.Write([]byte(name))
		hash.Write([]byte(value))
		hash.Write([]byte("\xff"))
	}
	return hash.Sum64()
}

func (ar *AlertingRule) newAlert(m datasource.Metric, ls *labelSet, start time.Time, qFn templates.QueryFn) (*notifier.Alert, error) {
	var err error
	if ls == nil {
		ls, err = ar.toLabels(m, qFn)
		if err != nil {
			return nil, fmt.Errorf("failed to expand labels: %s", err)
		}
	}
	a := &notifier.Alert{
		GroupID:  ar.GroupID,
		Name:     ar.Name,
		Labels:   ls.processed,
		Value:    m.Values[0],
		ActiveAt: start,
		Expr:     ar.Expr,
		For:      ar.For,
	}
	a.Annotations, err = a.ExecTemplate(qFn, ls.origin, ar.Annotations)
	return a, err
}

const (
	// alertMetricName is the metric name for synthetic alert timeseries.
	alertMetricName = "ALERTS"
	// alertForStateMetricName is the metric name for 'for' state of alert.
	alertForStateMetricName = "ALERTS_FOR_STATE"

	// alertNameLabel is the label name indicating the name of an alert.
	alertNameLabel = "alertname"
	// alertStateLabel is the label name indicating the state of an alert.
	alertStateLabel = "alertstate"

	// alertGroupNameLabel defines the label name attached for generated time series.
	// attaching this label may be disabled via `-disableAlertgroupLabel` flag.
	alertGroupNameLabel = "alertgroup"
)

// alertToTimeSeries converts the given alert with the given timestamp to time series
func (ar *AlertingRule) alertToTimeSeries(a *notifier.Alert, timestamp int64) []prompbmarshal.TimeSeries {
	var tss []prompbmarshal.TimeSeries
	tss = append(tss, alertToTimeSeries(a, timestamp))
	if ar.For > 0 {
		tss = append(tss, alertForToTimeSeries(a, timestamp))
	}
	return tss
}

func alertToTimeSeries(a *notifier.Alert, timestamp int64) prompbmarshal.TimeSeries {
	labels := make(map[string]string)
	for k, v := range a.Labels {
		labels[k] = v
	}
	labels["__name__"] = alertMetricName
	labels[alertStateLabel] = a.State.String()
	return newTimeSeries([]float64{1}, []int64{timestamp}, labels)
}

// alertForToTimeSeries returns a timeseries that represents
// state of active alerts, where value is time when alert become active
func alertForToTimeSeries(a *notifier.Alert, timestamp int64) prompbmarshal.TimeSeries {
	labels := make(map[string]string)
	for k, v := range a.Labels {
		labels[k] = v
	}
	labels["__name__"] = alertForStateMetricName
	return newTimeSeries([]float64{float64(a.ActiveAt.Unix())}, []int64{timestamp}, labels)
}

// restore restores the value of ActiveAt field for active alerts,
// based on previously written time series `alertForStateMetricName`.
// Only rules with For > 0 can be restored.
func (ar *AlertingRule) restore(ctx context.Context, q datasource.Querier, ts time.Time, lookback time.Duration) error {
	if ar.For < 1 {
		return nil
	}

	ar.AlertsMu.Lock()
	defer ar.AlertsMu.Unlock()

	if len(ar.Alerts) < 1 {
		return nil
	}

	for _, a := range ar.Alerts {
		if a.Restored || a.State != notifier.StatePending {
			continue
		}

		var labelsFilter []string
		for k, v := range a.Labels {
			labelsFilter = append(labelsFilter, fmt.Sprintf("%s=%q", k, v))
		}
		sort.Strings(labelsFilter)
		expr := fmt.Sprintf("last_over_time(%s{%s}[%ds])",
			alertForStateMetricName, strings.Join(labelsFilter, ","), int(lookback.Seconds()))

		ar.logDebugf(ts, nil, "restoring alert state via query %q", expr)

		res, _, err := q.Query(ctx, expr, ts)
		if err != nil {
			return err
		}

		qMetrics := res.Data
		if len(qMetrics) < 1 {
			ar.logDebugf(ts, nil, "no response was received from restore query")
			continue
		}

		// only one series expected in response
		m := qMetrics[0]
		// __name__ supposed to be alertForStateMetricName
		m.DelLabel("__name__")

		// we assume that restore query contains all label matchers,
		// so all received labels will match anyway if their number is equal.
		if len(m.Labels) != len(a.Labels) {
			ar.logDebugf(ts, nil, "state restore query returned not expected label-set %v", m.Labels)
			continue
		}
		a.ActiveAt = time.Unix(int64(m.Values[0]), 0)
		a.Restored = true
		logger.Infof("alert %q (%d) restored to state at %v", a.Name, a.ID, a.ActiveAt)
	}
	return nil
}

// alertsToSend walks through the current alerts of AlertingRule
// and returns only those which should be sent to notifier.
// Isn't concurrent safe.
func (ar *AlertingRule) alertsToSend(ts time.Time, resolveDuration, resendDelay time.Duration) []notifier.Alert {
	needsSending := func(a *notifier.Alert) bool {
		if a.State == notifier.StatePending {
			return false
		}
		if a.ResolvedAt.After(a.LastSent) {
			return true
		}
		return a.LastSent.Add(resendDelay).Before(ts)
	}

	var alerts []notifier.Alert
	for _, a := range ar.Alerts {
		if !needsSending(a) {
			continue
		}
		a.End = ts.Add(resolveDuration)
		if a.State == notifier.StateInactive {
			a.End = a.ResolvedAt
		}
		a.LastSent = ts
		alerts = append(alerts, *a)
	}
	return alerts
}
