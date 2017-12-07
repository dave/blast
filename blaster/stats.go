package blaster

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// Stats is a snapshot of the metrics (as is printed during interactive execution).
type Stats struct {
	ConcurrencyCurrent int
	ConcurrencyMaximum int
	Skipped            int64
	All                *Segment
	Segments           []*Segment
}

// Segment is a rate segment - a new segment is created each time the rate is changed.
type Segment struct {
	DesiredRate        float64
	ActualRate         float64
	AverageConcurrency float64
	Duration           time.Duration
	Summary            *Total
	Status             []*Status
}

// Total is the summary of all requests in this segment
type Total struct {
	Started     int64
	Finished    int64
	Success     int64
	Fail        int64
	Mean        time.Duration
	NinetyFifth time.Duration
}

// Status is a summary of all requests that returned a specific status
type Status struct {
	Status      string
	Count       int64
	Fraction    float64
	Mean        time.Duration
	NinetyFifth time.Duration
}

func (m *metricsDef) stats() Stats {
	m.sync.RLock()
	defer m.sync.RUnlock()

	s := Stats{
		All: &Segment{
			Summary: &Total{},
		},
	}

	// add some stats segments
	for range m.segments {
		s.Segments = append(s.Segments, &Segment{
			Summary: &Total{},
		})
	}

	// find all statuses and order
	var statuses []string
	for status := range m.all.status {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)

	for _, status := range statuses {
		s.All.Status = append(s.All.Status, &Status{Status: status})
		for _, seg := range s.Segments {
			seg.Status = append(seg.Status, &Status{Status: status})
		}
	}

	s.Skipped = m.skipped.Count()
	s.ConcurrencyCurrent = int(m.busy.Count())
	s.ConcurrencyMaximum = m.blaster.Workers
	s.All.ActualRate = float64(m.all.total.start.Count()) / m.all.duration().Seconds()
	s.All.AverageConcurrency = m.all.busy.Mean()
	s.All.Duration = m.all.duration()
	s.All.Summary.Started = m.all.total.start.Count()
	s.All.Summary.Finished = m.all.total.finish.Count()
	s.All.Summary.Success = m.all.total.success.Count()
	s.All.Summary.Fail = m.all.total.fail.Count()
	s.All.Summary.Mean = time.Duration(m.all.total.finish.Mean()/1000000.0) * time.Millisecond
	s.All.Summary.NinetyFifth = time.Duration(m.all.total.finish.Percentile(0.95)/1000000.0) * time.Millisecond

	for i, seg := range s.Segments {
		seg.DesiredRate = m.segments[i].rate
		seg.ActualRate = float64(m.segments[i].total.start.Count()) / m.segments[i].duration().Seconds()
		seg.AverageConcurrency = m.segments[i].busy.Mean()
		seg.Duration = m.segments[i].duration()
		seg.Summary.Started = m.segments[i].total.start.Count()
		seg.Summary.Finished = m.segments[i].total.finish.Count()
		seg.Summary.Success = m.segments[i].total.success.Count()
		seg.Summary.Fail = m.segments[i].total.fail.Count()
		seg.Summary.Mean = time.Duration(m.segments[i].total.finish.Mean()/1000000.0) * time.Millisecond
		seg.Summary.NinetyFifth = time.Duration(m.segments[i].total.finish.Percentile(0.95)/1000000.0) * time.Millisecond
	}

	for statusIndex, status := range statuses {
		s.All.Status[statusIndex].Count = m.all.status[status].finish.Count()
		s.All.Status[statusIndex].Fraction = float64(m.all.status[status].finish.Count()) / float64(m.all.total.finish.Count())
		s.All.Status[statusIndex].Mean = time.Duration(m.all.status[status].finish.Mean()/1000000.0) * time.Millisecond
		s.All.Status[statusIndex].NinetyFifth = time.Duration(m.all.status[status].finish.Percentile(0.95)/1000000.0) * time.Millisecond
		for segmentIndex, seg := range s.Segments {
			if m.segments[segmentIndex].status[status] != nil {
				seg.Status[statusIndex].Count = m.segments[segmentIndex].status[status].finish.Count()
				seg.Status[statusIndex].Fraction = float64(m.segments[segmentIndex].status[status].finish.Count()) / float64(m.segments[segmentIndex].total.finish.Count())
				seg.Status[statusIndex].Mean = time.Duration(m.segments[segmentIndex].status[status].finish.Mean()/1000000.0) * time.Millisecond
				seg.Status[statusIndex].NinetyFifth = time.Duration(m.segments[segmentIndex].status[status].finish.Percentile(0.95)/1000000.0) * time.Millisecond
			}
		}
	}

	return s
}

// String returns a string representation of the stats (as is printed during interactive execution).
func (s Stats) String() string {
	buf := &bytes.Buffer{}
	w := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "Metrics")
	fmt.Fprintln(w, "=======")

	var segments []int
	for i := len(s.Segments) - 1; i >= 0; i-- {
		segments = append(segments, i)
	}

	tabs := strings.Repeat("\t", len(segments)+2)

	if s.Skipped > 0 {
		fmt.Fprintf(w, "Skipped:\t%d from previous runs\n", s.Skipped)
	}

	fmt.Fprintf(w, "Concurrency:\t%d / %d workers in use\n", s.ConcurrencyCurrent, s.ConcurrencyMaximum)
	fmt.Fprintf(w, "%s\n", tabs)

	fmt.Fprint(w, "Desired rate:\t(all)\t")
	for _, i := range segments {
		fmt.Fprintf(w, "%.0f\t", s.Segments[i].DesiredRate)
	}
	fmt.Fprintf(w, "%s\n", tabs)

	fmt.Fprint(w, "Actual rate:\t")
	fmt.Fprintf(w, "%.0f\t", s.All.ActualRate)
	for _, i := range segments {
		fmt.Fprintf(w, "%.0f\t", s.Segments[i].ActualRate)
	}
	fmt.Fprintf(w, "%s\n", tabs)

	fmt.Fprint(w, "Avg concurrency:\t")
	fmt.Fprintf(w, "%.0f\t", s.All.AverageConcurrency)
	for _, i := range segments {
		fmt.Fprintf(w, "%.0f\t", s.Segments[i].AverageConcurrency)
	}
	fmt.Fprintf(w, "%s\n", tabs)

	fmt.Fprint(w, "Duration:\t")
	fmt.Fprintf(w, "%s\t", fmtDuration(s.All.Duration))
	for _, i := range segments {
		fmt.Fprintf(w, "%s\t", fmtDuration(s.Segments[i].Duration))
	}
	fmt.Fprintf(w, "%s\n", tabs)

	fmt.Fprintf(w, "%s\n", tabs)
	fmt.Fprintf(w, "Total%s\n", tabs)
	fmt.Fprintf(w, "-----%s\n", tabs)

	fmt.Fprintf(w, "Started:\t%d\t", s.All.Summary.Started)
	for _, i := range segments {
		fmt.Fprintf(w, "%d\t", s.Segments[i].Summary.Started)
	}
	fmt.Fprint(w, "\n")
	fmt.Fprintf(w, "Finished:\t%d\t", s.All.Summary.Finished)
	for _, i := range segments {
		fmt.Fprintf(w, "%d\t", s.Segments[i].Summary.Finished)
	}
	fmt.Fprint(w, "\n")
	fmt.Fprintf(w, "Success:\t%d\t", s.All.Summary.Success)
	for _, i := range segments {
		fmt.Fprintf(w, "%d\t", s.Segments[i].Summary.Success)
	}
	fmt.Fprint(w, "\n")
	fmt.Fprintf(w, "Fail:\t%d\t", s.All.Summary.Fail)
	for _, i := range segments {
		fmt.Fprintf(w, "%d\t", s.Segments[i].Summary.Fail)
	}
	fmt.Fprint(w, "\n")
	fmt.Fprintf(w, "Mean:\t%.1f ms\t", s.All.Summary.Mean.Seconds()*1000)
	for _, i := range segments {
		fmt.Fprintf(w, "%.1f ms\t", s.Segments[i].Summary.Mean.Seconds()*1000)
	}
	fmt.Fprintf(w, "%s\n", tabs)
	fmt.Fprintf(w, "95th:\t%.1f ms\t", s.All.Summary.NinetyFifth.Seconds()*1000)
	for _, i := range segments {
		fmt.Fprintf(w, "%.1f ms\t", s.Segments[i].Summary.NinetyFifth.Seconds()*1000)
	}
	fmt.Fprintf(w, "%s\n", tabs)

	for status := range s.All.Status {

		fmt.Fprintf(w, "%s\n", tabs)
		fmt.Fprintf(w, "%v%s\n", s.All.Status[status].Status, tabs)
		fmt.Fprintf(w, "%s%s\n", strings.Repeat("-", len(s.All.Status[status].Status)), tabs)

		fmt.Fprintf(w, "Count:\t%d (%.0f%%)\t", s.All.Status[status].Count, 100*s.All.Status[status].Fraction)
		for _, i := range segments {
			if s.Segments[i].Status[status].Count == 0 {
				fmt.Fprint(w, "0\t")
			} else {
				fmt.Fprintf(w, "%d (%.0f%%)\t", s.Segments[i].Status[status].Count, 100*s.Segments[i].Status[status].Fraction)
			}
		}
		fmt.Fprint(w, "\n")

		fmt.Fprintf(w, "Mean:\t%.1f ms\t", s.All.Status[status].Mean.Seconds()*1000)
		for _, i := range segments {
			if s.Segments[i].Status[status].Count == 0 {
				fmt.Fprint(w, "-\t")
			} else {
				fmt.Fprintf(w, "%.1f ms\t", s.Segments[i].Status[status].Mean.Seconds()*1000)
			}
		}
		fmt.Fprintf(w, "%s\n", tabs)
		fmt.Fprintf(w, "95th:\t%.1f ms\t", s.All.Status[status].NinetyFifth.Seconds()*1000)
		for _, i := range segments {
			if s.Segments[i].Status[status].Count == 0 {
				fmt.Fprint(w, "-\t")
			} else {
				fmt.Fprintf(w, "%.1f ms\t", s.Segments[i].Status[status].NinetyFifth.Seconds()*1000)
			}
		}
		fmt.Fprintf(w, "%s\n", tabs)
	}
	w.Flush()
	return buf.String()
}

func fmtDuration(d time.Duration) string {
	sec := int(d.Seconds())
	min := sec / 60
	hr := min / 60
	if hr > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hr, min%60, sec%60)
	}
	return fmt.Sprintf("%02d:%02d", min%60, sec%60)
}
