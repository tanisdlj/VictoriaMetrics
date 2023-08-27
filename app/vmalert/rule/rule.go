package rule

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/prompbmarshal"
)

// Rule represents alerting or recording rule
// that has unique ID, can be Executed and
// updated with other Rule.
type Rule interface {
	// ID returns unique ID that may be used for
	// identifying this Rule among others.
	ID() uint64
	// exec executes the rule with given context at the given timestamp and limit.
	// returns an err if number of resulting time series exceeds the limit.
	exec(ctx context.Context, ts time.Time, limit int) ([]prompbmarshal.TimeSeries, error)
	// execRange executes the rule on the given time range.
	execRange(ctx context.Context, start, end time.Time) ([]prompbmarshal.TimeSeries, error)
	// updateWith performs modification of current Rule
	// with fields of the given Rule.
	updateWith(Rule) error
	// close performs the shutdown procedures for rule
	// such as metrics unregister
	close()
}

var errDuplicate = errors.New("result contains metrics with the same labelset after applying rule labels. See https://docs.victoriametrics.com/vmalert.html#series-with-the-same-labelset for details")

type ruleState struct {
	sync.RWMutex
	entries []StateEntry
	cur     int
}

// StateEntry stores rule's execution states
type StateEntry struct {
	// stores last moment of time rule.Exec was called
	Time time.Time
	// stores the timesteamp with which rule.Exec was called
	At time.Time
	// stores the duration of the last rule.Exec call
	Duration time.Duration
	// stores last error that happened in Exec func
	// resets on every successful Exec
	// may be used as Health ruleState
	Err error
	// stores the number of samples returned during
	// the last evaluation
	Samples int
	// stores the number of time series fetched during
	// the last evaluation.
	// Is supported by VictoriaMetrics only, starting from v1.90.0
	// If seriesFetched == nil, then this attribute was missing in
	// datasource response (unsupported).
	SeriesFetched *int
	// stores the curl command reflecting the HTTP request used during rule.Exec
	Curl string
}

// NewRuleState create ruleState with given size RuleStateEntry
func NewRuleState(size int) *ruleState {
	if size < 1 {
		size = 1
	}
	return &ruleState{
		entries: make([]StateEntry, size),
	}
}

func (s *ruleState) GetLast() StateEntry {
	s.RLock()
	defer s.RUnlock()
	return s.entries[s.cur]
}

func (s *ruleState) Size() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.entries)
}

func (s *ruleState) GetAll() []StateEntry {
	entries := make([]StateEntry, 0)

	s.RLock()
	defer s.RUnlock()

	cur := s.cur
	for {
		e := s.entries[cur]
		if !e.Time.IsZero() || !e.At.IsZero() {
			entries = append(entries, e)
		}
		cur--
		if cur < 0 {
			cur = cap(s.entries) - 1
		}
		if cur == s.cur {
			return entries
		}
	}
}

func (s *ruleState) Add(e StateEntry) {
	s.Lock()
	defer s.Unlock()

	s.cur++
	if s.cur > cap(s.entries)-1 {
		s.cur = 0
	}
	s.entries[s.cur] = e
}
