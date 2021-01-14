package main

import (
	"fmt"
	"github.com/matryer/is"
	"testing"
)

func TestCheckVersion(t *testing.T) {
	tt := []struct {
		name     string
		majorVer int
		minorVer int
		err      error
	}{
		{
			name:     "lower major version",
			majorVer: 7,
			minorVer: 2,
			err:      fmt.Errorf("Install libips=>'8.9'. Current version is 7.2"),
		},
		{
			name:     "lower minor version",
			majorVer: 8,
			minorVer: 5,
			err:      fmt.Errorf("Install libips=>'8.9'. Current version is 8.5"),
		},
		{
			name:     "equal version",
			majorVer: 8,
			minorVer: 9,
			err:      nil,
		},
		{
			name:     "higher version",
			majorVer: 9,
			minorVer: 2,
			err:      nil,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("TestCheckVersionFunction: %s", tc.name), func(t *testing.T) {
			is := is.NewRelaxed(t)
			err := checkVipsVersion(tc.majorVer, tc.minorVer)
			is.Equal(err, tc.err)
		})
	}
}
