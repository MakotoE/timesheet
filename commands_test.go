package timesheet

import (
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/MakotoE/checkerror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_dailyDurations(t *testing.T) {
	tests := []struct {
		entries  []entry
		expected []time.Duration
	}{
		{
			nil,
			nil,
		},
		{
			[]entry{{
				date:     time.Date(0, 1, 2, 0, 0, 0, 0, time.UTC),
				duration: time.Duration(1),
			}},
			[]time.Duration{time.Duration(1)},
		},
		{
			[]entry{
				{
					date:     time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC),
					duration: time.Duration(1),
				},
				{
					date:     time.Date(0, 0, 0, 0, 0, 0, 1, time.UTC),
					duration: time.Duration(2),
				},
			},
			[]time.Duration{time.Duration(3)},
		},
		{
			[]entry{
				{
					date:     time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC),
					duration: time.Duration(1),
				},
				{
					date:     time.Date(0, 0, 1, 0, 0, 0, 0, time.UTC),
					duration: time.Duration(2),
				},
			},
			[]time.Duration{time.Duration(1), time.Duration(2)},
		},
	}

	for i, test := range tests {
		assert.Equal(t, test.expected, dailyDurations(test.entries), i)
	}
}

func TestWeeklyTotal(t *testing.T) {
	tests := []struct {
		durations []time.Duration
		expected  time.Duration
	}{
		{
			nil,
			0,
		},
		{
			[]time.Duration{1},
			1,
		},
		{
			[]time.Duration{1, 2},
			3,
		},
	}

	for i, test := range tests {
		assert.Equal(t, test.expected, weeklyTotal(test.durations), i)
	}
}

func TestSundayIndex(t *testing.T) {
	tests := []struct {
		entry0   entry
		expected int
	}{
		{
			entry{date: time.Date(2019, 8, 11, 0, 0, 0, 0, time.UTC)}, // Sunday
			0,
		},
		{
			entry{date: time.Date(2019, 8, 10, 0, 0, 0, 0, time.UTC)}, // Saturday
			1,
		},
		{
			entry{date: time.Date(2019, 8, 12, 0, 0, 0, 0, time.UTC)}, // Monday
			6,
		},
	}

	for i, test := range tests {
		assert.Equal(t, test.expected, sundayIndex(test.entry0), i)
	}
}

func Test_nextLogRecord(t *testing.T) {
	testDate := time.Date(1, 2, 3, 4, 5, 6, 7, time.UTC)
	testDateText, err := testDate.MarshalText()
	require.Nil(t, err)

	tests := []struct {
		text        string
		expected    *entry
		expectError bool
	}{
		{
			"",
			nil,
			true,
		},
		{
			"a,a",
			nil,
			true,
		},
		{
			string(testDateText) + "," + time.Duration(1).String(),
			&entry{testDate, time.Duration(1)},
			false,
		},
	}

	for i, test := range tests {
		result, err := nextLogRecord(csv.NewReader(strings.NewReader(test.text)))
		if test.expected == nil {
			assert.Equal(t, test.expected, result, i)
		} else {
			assert.True(t, test.expected.date.Equal(result.date), i)
			assert.Equal(t, test.expected.duration, result.duration, i)
		}
		checkerror.Check(t, test.expectError, err, i)
	}
}
