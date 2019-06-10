package main

import (
	"sort"
	"testing"

	"github.com/brotherlogic/goserver/utils"
	pbrc "github.com/brotherlogic/recordcollection/proto"
)

var data = []struct {
	in  []*pbrc.Record
	out []*pbrc.Record
}{
	{
		// NOT_KEEPER should come before KEEPER
		[]*pbrc.Record{
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_KEEPER}},
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_NOT_KEEPER}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_NOT_KEEPER}},
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_KEEPER}},
		},
	}}

func TestOrdering(t *testing.T) {
	for _, entry := range data {
		sort.Sort(ByReleaseDate(entry.in))
		for i := range entry.in {
			if !utils.FuzzyMatch(entry.in[i], entry.out[i]) {
				t.Errorf("Sorting error: %v vs %v", entry.in[i], entry.out[i])
			}
		}
	}
}
