package library

import (
	"errors"
	"time"
)

// jakarta is the zone in which a grant's end date is read. Indonesia's western time has no daylight saving,
// so a fixed offset is exact and needs no timezone database on the host.
var jakarta = time.FixedZone("Asia/Jakarta", 7*60*60)

// ErrInvalidEndDate is returned for an end date that is not a real calendar day in YYYY-MM-DD form.
var ErrInvalidEndDate = errors.New("library: end date must be a real day in YYYY-MM-DD form")

const endDateLayout = "2006-01-02"

// endExclusive returns the first instant after the last day of access: midnight at the start of the day
// following endsOn in Asia/Jakarta.
func endExclusive(endsOn string) (time.Time, error) {
	day, err := time.ParseInLocation(endDateLayout, endsOn, jakarta)
	if err != nil {
		return time.Time{}, ErrInvalidEndDate
	}
	return day.AddDate(0, 0, 1), nil
}

// AccessEndsAt returns the last second of access for an end date: 23:59:59 Asia/Jakarta of that day.
func AccessEndsAt(endsOn string) (time.Time, error) {
	end, err := endExclusive(endsOn)
	if err != nil {
		return time.Time{}, err
	}
	return end.Add(-time.Second), nil
}

// AccessOpenAt reports whether access that ends on endsOn is still open at now. Access stays open through
// the whole of the end date, until midnight Asia/Jakarta begins the next day.
func AccessOpenAt(endsOn string, now time.Time) (bool, error) {
	end, err := endExclusive(endsOn)
	if err != nil {
		return false, err
	}
	return now.Before(end), nil
}
