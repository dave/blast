package blaster

import (
	"testing"
	"time"
)

func TestStats_String(t *testing.T) {
	s := Stats{
		ConcurrencyCurrent: 1,
		ConcurrencyMaximum: 2,
		Skipped:            3,
		All: &Segment{
			DesiredRate:        4,
			ActualRate:         5,
			AverageConcurrency: 6,
			Duration:           time.Second * 7,
			Summary: &Total{
				Started:     8,
				Finished:    9,
				Success:     10,
				Fail:        11,
				Mean:        time.Millisecond * 12,
				NinetyFifth: time.Millisecond * 13,
			},
			Status: []*Status{
				{
					Status:      "a",
					Count:       14,
					Fraction:    0.15,
					Mean:        time.Millisecond * 16,
					NinetyFifth: time.Millisecond * 17,
				},
				{
					Status:      "b",
					Count:       18,
					Fraction:    0.19,
					Mean:        time.Millisecond * 20,
					NinetyFifth: time.Millisecond * 21,
				},
			},
		},
		Segments: []*Segment{
			{
				DesiredRate:        22,
				ActualRate:         23,
				AverageConcurrency: 24,
				Duration:           time.Second * 125,
				Summary: &Total{
					Started:     26,
					Finished:    27,
					Success:     28,
					Fail:        29,
					Mean:        time.Millisecond * 30,
					NinetyFifth: time.Millisecond * 31,
				},
				Status: []*Status{
					{
						Status:      "a",
						Count:       32,
						Fraction:    0.33,
						Mean:        time.Millisecond * 34,
						NinetyFifth: time.Millisecond * 35,
					},
					{
						Status:      "b",
						Count:       36,
						Fraction:    0.37,
						Mean:        time.Millisecond * 38,
						NinetyFifth: time.Millisecond * 38,
					},
				},
			},
			{
				DesiredRate:        39,
				ActualRate:         40,
				AverageConcurrency: 41,
				Duration:           time.Second * 4042,
				Summary: &Total{
					Started:     43,
					Finished:    44,
					Success:     45,
					Fail:        46,
					Mean:        time.Millisecond * 47,
					NinetyFifth: time.Millisecond * 48,
				},
				Status: []*Status{
					{
						Status:      "a",
						Count:       49,
						Fraction:    0.50,
						Mean:        time.Millisecond * 51,
						NinetyFifth: time.Millisecond * 52,
					},
					{
						Status:      "b",
						Count:       0,
						Fraction:    0,
						Mean:        0,
						NinetyFifth: 0,
					},
				},
			},
		},
	}
	expected := `Metrics
=======
Skipped:          3 from previous runs
Concurrency:      1 / 2 workers in use
                                                
Desired rate:     (all)     39        22                
Actual rate:      5         40        23                
Avg concurrency:  6         41        24                
Duration:         00:07     1:07:22   02:05             
                                                
Total                                           
-----                                           
Started:          8         43        26        
Finished:         9         44        27        
Success:          10        45        28        
Fail:             11        46        29        
Mean:             12.0 ms   47.0 ms   30.0 ms           
95th:             13.0 ms   48.0 ms   31.0 ms           
                                                
a                                               
-                                               
Count:            14 (15%)  49 (50%)  32 (33%)  
Mean:             16.0 ms   51.0 ms   34.0 ms           
95th:             17.0 ms   52.0 ms   35.0 ms           
                                                
b                                               
-                                               
Count:            18 (19%)  0         36 (37%)  
Mean:             20.0 ms   -         38.0 ms           
95th:             21.0 ms   -         38.0 ms           
`
	if s.String() != expected {
		t.Fatal("Unexpected stat string:", s.String())
	}
}
