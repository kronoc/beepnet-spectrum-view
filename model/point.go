package model

import (
	"fmt"
	"regexp"
	"strconv"
)

type Point [2]float64

var re *regexp.Regexp = regexp.MustCompile(`\(([-+]?[0-9]*\.?[0-9]+),([-+]?[0-9]*\.?[0-9]+)\)`)

func (p Point) String() string {
	return fmt.Sprintf("(%f,%f)", p[0], p[1])
}

func StringToPoint(s string, p *Point) error {
	parseError := func() error {
		return fmt.Errorf("Invalid Point string: %q", s)
	}

	result := re.FindStringSubmatch(s)

	if len(result) != 3 {
		return parseError()
	}

	x, err := strconv.ParseFloat(result[1], 64)
	y, err := strconv.ParseFloat(result[2], 64)

	if err != nil {
		return parseError()
	}

	p[0], p[1] = x, y
	return nil
}

func NewPoint(s string) (*Point, error) {
	point := new(Point)

	if err := StringToPoint(s, point); err != nil {
		return nil, err
	}

	return point, nil
}
