package model

import (
	"database/sql"
	"fmt"
)

// NOTE: This package assumes the database is postgres
/**********************************
CREATE TABLE IF NOT EXISTS sample (
    id         bigserial PRIMARY KEY,
    power      numeric NOT NULL,
    freq       bigint NOT NULL,
    bandwidth  integer NOT NULL,
    decfactor  integer NOT NULL DEFAULT 1,
    survey_id  integer REFERENCES survey
);
**********************************/

func DefaultDecimationFactors() []int {
	return []int{100, 200, 400}
}

type Sample struct {
	Power     float64 `json:"power"`
	Freq      uint64  `json:"freq"`
	Bandwidth uint32  `json:"bandwidth"`
}

func GetSamplesForSurveyId(id int, df int, db *sql.DB) ([]Sample, error) {
	var smps []Sample

	rows, err := db.Query(`
			SELECT power, freq, bandwidth
			FROM sample
			WHERE survey_id = $1 AND decfactor = $2
			ORDER BY freq`, id, df)
	if err != nil {
		return nil, fmt.Errorf(`DB Error: %q`, err)
	}

	for rows.Next() {
		var smp Sample
		if err := rows.Scan(&smp.Power, &smp.Freq, &smp.Bandwidth); err != nil {
			return nil, fmt.Errorf(`DB error: %q`, err)
		}
		smps = append(smps, smp)
	}

	return smps, nil
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
