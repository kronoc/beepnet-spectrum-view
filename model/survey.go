package model

import (
	"database/sql"
	"fmt"
	"time"
)

// NOTE: This package assumes the database is postgres
/**********************************
CREATE TABLE IF NOT EXISTS survey (
    id       serial PRIMARY KEY,
    label    varchar(256) NOT NULL,
    location point NOT NULL,
    time     timestamp NOT NULL,
    raw_data text NOT NULL
);
**********************************/

type Survey struct {
	Id       int       `json:"id"`
	Label    string    `json:"label"`
	Location Point     `json:"location"`
	Time     time.Time `json:"time"`
	RawData  string    `json:"-"`
	Tags     TagsType  `json:"tags"`
}

// WriteToDB should be
func (s *Survey) WriteToDB(tx *sql.Tx) (int, error) {
	var surveyId int
	if err := tx.QueryRow(`
			INSERT INTO survey
			(label, location, time, raw_data)
			VALUES ($1, $2, $3, $4)
			RETURNING id`,
		s.Label, s.Location.String(), s.Time, s.RawData).Scan(&surveyId); err != nil {
		return -1, fmt.Errorf(`DB Error: %q`, err)
	}

	return surveyId, nil
}

func GetSurveys(db *sql.DB) ([]Survey, error) {
	var svs []Survey

	rows, err := db.Query(`
		SELECT id, label, location, time
		FROM sv`)
	if err != nil {
		return nil, fmt.Errorf(`DB Error: %q`, err)
	}

	for rows.Next() {
		var sv Survey
		var locbuf []byte
		if err := rows.Scan(&sv.Id, &sv.Label, &locbuf, &sv.Time); err != nil {
			return nil, fmt.Errorf(`DB Error: %q`, err)
		}

		if err := StringToPoint(string(locbuf), &sv.Location); err != nil {
			return nil, fmt.Errorf(`Error parsing location: %q`, err)
		}

		svs = append(svs, sv)
	}

	return svs, nil
}

func GetSurveyById(id int, db *sql.DB) (*Survey, error) {
	var sv *Survey

	row := db.QueryRow(`
		SELECT id, label, location, time
		FROM sv
		WHERE id = $1`, id)

	var locbuf []byte
	if err := row.Scan(&sv.Id, &sv.Label, &locbuf, &sv.Time); err != nil {
		return nil, fmt.Errorf(`DB Error: %q`, err)
	}

	if err := StringToPoint(string(locbuf), &sv.Location); err != nil {
		return nil, fmt.Errorf(`Error parsing location: %q`, err)
	}

	return sv, nil
}

func (s *Survey) String() string {
	return fmt.Sprintf("%q (%d) @ (%f,%f), %s (tags: %s)",
		s.Label, s.Id, s.Location[0], s.Location[1], s.Time, s.Tags)
}
