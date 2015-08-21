package model

import (
	"fmt"
	"time"
)

type Survey struct {
	Id       int       `json:"id"`
	Label    string    `json:"label"`
	Location Point     `json:"location"`
	Time     time.Time `json:"time"`
}

func (s *Survey) String() string {
	return fmt.Sprintf("%q (%d) @ (%f,%f), %s",
		s.Label, s.Id, s.Location[0], s.Location[1], s.Time)
}

type Sample struct {
	Power     float64 `json:"power"`
	Freq      uint64  `json:"freq"`
	Bandwidth uint32  `json:"bandwidth"`
}

func (s *Sample) String() string {
	return fmt.Sprintf("%.4fdb @ %d +/- %dHz", s.Power, s.Freq, s.Bandwidth/2)
}

// SampleVector is a simplified representation of Sample that allows for more
// concise JSON representation (e.g., in the upload handler)
type SampleVector [3]float64

func (s SampleVector) ToSample() *Sample {
	return &Sample{s[0], uint64(s[1]), uint32(s[2])}
}

func (s SampleVector) String() string {
	return s.ToSample().String()
}
