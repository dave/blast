package blaster

import (
	"sync"
	"time"

	"fmt"
	"io"

	"sort"

	"strings"

	"runtime"

	"github.com/rcrowley/go-metrics"
)

type metricsDef struct {
	sync     sync.RWMutex
	registry metrics.Registry
	current  int
	skipped  metrics.Counter
	busy     metrics.Counter
	all      *metricsSegment
	segments []*metricsSegment
	blaster  *Blaster
}

func newMetricsDef(b *Blaster) *metricsDef {
	r := metrics.NewRegistry()
	m := &metricsDef{
		registry: r,
		busy:     metrics.NewRegisteredCounter("busy", r),
		skipped:  metrics.NewRegisteredCounter("skipped", r),
		blaster:  b,
	}
	m.all = m.newMetricsSegment(0)
	return m
}

func (m *metricsDef) logMiss() {
	m.sync.RLock()
	defer m.sync.RUnlock()
	m.all.missed.Inc(1)
	m.segments[m.current].missed.Inc(1)
}

func (m *metricsDef) logBusy(segment int) {
	m.sync.RLock()
	defer m.sync.RUnlock()
	m.all.busy.Update(m.busy.Count())
	m.segments[segment].busy.Update(m.busy.Count())
}

func (m *metricsDef) logSkip() {
	m.skipped.Inc(1)
}

func (m *metricsDef) currentSegment() int {
	m.sync.RLock()
	defer m.sync.RUnlock()
	return m.current
}

func (m *metricsDef) logStart(segment int) {
	m.sync.RLock()
	defer m.sync.RUnlock()
	m.all.logStart()
	m.segments[segment].logStart()
}

func (m *metricsDef) logFinish(segment int, status string, elapsed time.Duration, success bool) {
	m.sync.RLock()
	defer m.sync.RUnlock()
	m.all.logFinish(status, elapsed, success)
	m.segments[segment].logFinish(status, elapsed, success)
}

func (m *metricsDef) addSegment(rate float64) {
	m.sync.Lock()
	defer m.sync.Unlock()
	if len(m.segments) > 0 {
		m.segments[m.current].end = time.Now()
	}
	m.segments = append(m.segments, m.newMetricsSegment(rate))
	m.current = len(m.segments) - 1
}

func (m *metricsDef) newMetricsItem() *metricsItem {
	return &metricsItem{
		start:   metrics.NewRegisteredCounter("start", m.registry),
		finish:  metrics.NewRegisteredTimer("finish", m.registry),
		success: metrics.NewRegisteredCounter("success", m.registry),
		fail:    metrics.NewRegisteredCounter("fail", m.registry),
	}
}

func (m *metricsDef) newMetricsSegment(rate float64) *metricsSegment {
	return &metricsSegment{
		def:    m,
		rate:   rate,
		total:  m.newMetricsItem(),
		status: map[string]*metricsItem{},
		missed: metrics.NewRegisteredCounter("missed", m.registry),
		busy:   metrics.NewRegisteredHistogram("busy", m.registry, metrics.NewExpDecaySample(1028, 0.015)),
		start:  time.Now(),
	}
}

type metricsSegment struct {
	sync   sync.RWMutex
	def    *metricsDef
	rate   float64
	missed metrics.Counter
	busy   metrics.Histogram
	total  *metricsItem
	status map[string]*metricsItem
	start  time.Time
	end    time.Time
}

func (m *metricsSegment) duration() time.Duration {
	if m.end == (time.Time{}) {
		return time.Since(m.start)
	} else {
		return m.end.Sub(m.start)
	}
}

func (m *metricsSegment) logStart() {
	m.total.start.Inc(1)
}

func (m *metricsSegment) logFinish(status string, elapsed time.Duration, success bool) {
	m.sync.Lock()
	defer m.sync.Unlock()

	if _, ok := m.status[status]; !ok {
		m.status[status] = m.def.newMetricsItem()
	}

	m.total.finish.Update(elapsed)
	m.status[status].finish.Update(elapsed)

	if success {
		m.total.success.Inc(1)
		m.status[status].success.Inc(1)
	} else {
		m.total.fail.Inc(1)
		m.status[status].fail.Inc(1)
	}
}

type metricsItem struct {
	start   metrics.Counter
	finish  metrics.Timer
	success metrics.Counter
	fail    metrics.Counter
}

func (m *metricsDef) summary(w io.Writer) {
	m.sync.RLock()
	defer m.sync.RUnlock()
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Metrics")
	fmt.Fprintln(w, "=======")

	var segments []int
	for i := len(m.segments) - 1; i >= 0; i-- {
		segments = append(segments, i)
	}

	// find all statuses and order
	var statuses []string
	for status, _ := range m.all.status {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)

	tabs := strings.Repeat("\t", len(segments)+2)

	if m.skipped.Count() > 0 {
		fmt.Fprintf(w, "Skipped:\t%d from previous runs\n", m.skipped.Count())
	}

	fmt.Fprintf(w, "Concurrency:\t%d / %d workers in use\n", m.busy.Count(), m.blaster.Workers)
	fmt.Fprintf(w, "%s\n", tabs)

	if DEBUG {
		fmt.Fprintf(w, "Goroutines:\t%d\n", runtime.NumGoroutine())
		fmt.Fprintf(w, "%s\n", tabs)
	}

	fmt.Fprint(w, "Desired rate:\t(all)\t")
	for _, i := range segments {
		fmt.Fprintf(w, "%.0f\t", m.segments[i].rate)
	}
	fmt.Fprintf(w, "%s\n", tabs)

	fmt.Fprint(w, "Actual rate:\t")
	fmt.Fprintf(w, "%.0f\t", float64(m.all.total.start.Count())/m.all.duration().Seconds())
	for _, i := range segments {
		fmt.Fprintf(w, "%.0f\t", float64(m.segments[i].total.start.Count())/m.segments[i].duration().Seconds())
	}
	fmt.Fprintf(w, "%s\n", tabs)

	fmt.Fprint(w, "Avg concurrency:\t")
	fmt.Fprintf(w, "%.0f\t", m.all.busy.Mean())
	for _, i := range segments {
		fmt.Fprintf(w, "%.0f\t", m.segments[i].busy.Mean())
	}
	fmt.Fprintf(w, "%s\n", tabs)

	fmt.Fprint(w, "Duration:\t")
	fmt.Fprintf(w, "%s\t", fmtDuration(m.all.duration()))
	for _, i := range segments {
		fmt.Fprintf(w, "%s\t", fmtDuration(m.segments[i].duration()))
	}
	fmt.Fprintf(w, "%s\n", tabs)

	fmt.Fprintf(w, "%s\n", tabs)
	fmt.Fprintf(w, "Total%s\n", tabs)
	fmt.Fprintf(w, "-----%s\n", tabs)
	m.printRows(w, true, segments, tabs, m.all.total, func(i int) *metricsItem { return m.segments[i].total })

	for _, status := range statuses {
		fmt.Fprintf(w, "%s\n", tabs)
		fmt.Fprintf(w, "%v%s\n", status, tabs)
		fmt.Fprintf(w, "%s%s\n", strings.Repeat("-", len(status)), tabs)
		m.printRows(w, false, segments, tabs, m.all.status[status], func(i int) *metricsItem { return m.segments[i].status[status] })
	}
}

func fmtDuration(d time.Duration) string {
	sec := int(d.Seconds())
	min := sec / 60
	hr := min / 60
	if hr > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hr, min%60, sec%60)
	} else {
		return fmt.Sprintf("%02d:%02d", min%60, sec%60)
	}
}

func (m *metricsDef) printRows(w io.Writer, all bool, segments []int, tabs string, total *metricsItem, status func(i int) *metricsItem) {
	if all {
		fmt.Fprintf(w, "Started:\t%d\t", total.start.Count())
		for _, i := range segments {
			if status(i) == nil {
				fmt.Fprint(w, "0\t")
			} else {
				fmt.Fprintf(w, "%d\t", status(i).start.Count())
			}
		}
		fmt.Fprint(w, "\n")
		fmt.Fprintf(w, "Finished:\t%d\t", total.finish.Count())
		for _, i := range segments {
			if status(i) == nil {
				fmt.Fprint(w, "0\t")
			} else {
				fmt.Fprintf(w, "%d\t", status(i).finish.Count())
			}
		}
		fmt.Fprint(w, "\n")
		fmt.Fprintf(w, "Success:\t%d\t", total.success.Count())
		for _, i := range segments {
			if status(i) == nil {
				fmt.Fprint(w, "0\t")
			} else {
				fmt.Fprintf(w, "%d\t", status(i).success.Count())
			}
		}
		fmt.Fprint(w, "\n")
		fmt.Fprintf(w, "Fail:\t%d\t", total.fail.Count())
		for _, i := range segments {
			if status(i) == nil {
				fmt.Fprint(w, "0\t")
			} else {
				fmt.Fprintf(w, "%d\t", status(i).fail.Count())
			}
		}
		fmt.Fprint(w, "\n")
	} else {
		fmt.Fprintf(w, "Count:\t%d (%.0f%%)\t", total.finish.Count(), 100*float64(total.finish.Count())/float64(m.all.total.finish.Count()))
		for _, i := range segments {
			if status(i) == nil {
				fmt.Fprint(w, "0\t")
			} else {
				fmt.Fprintf(w, "%d (%.0f%%)\t", status(i).finish.Count(), 100*float64(status(i).finish.Count())/float64(m.segments[i].total.finish.Count()))
			}
		}
		fmt.Fprint(w, "\n")
	}
	fmt.Fprintf(w, "Mean:\t%.1f ms\t", total.finish.Mean()/1000000)
	for _, i := range segments {
		if status(i) == nil {
			fmt.Fprint(w, "-\t")
		} else {
			fmt.Fprintf(w, "%.1f ms\t", status(i).finish.Mean()/1000000)
		}
	}
	fmt.Fprintf(w, "%s\n", tabs)
	fmt.Fprintf(w, "95th:\t%.1f ms\t", total.finish.Percentile(0.95)/1000000)
	for _, i := range segments {
		if status(i) == nil {
			fmt.Fprint(w, "-\t")
		} else {
			fmt.Fprintf(w, "%.1f ms\t", status(i).finish.Percentile(0.95)/1000000)
		}
	}
	fmt.Fprintf(w, "%s\n", tabs)
}
