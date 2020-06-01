package main

import (
	"sort"
	"testing"
	"time"

	"github.com/brotherlogic/goserver/utils"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
	"github.com/brotherlogic/recordsorganiser/sales"
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
		sort.Sort(sales.BySaleOrder(entry.in))
		for i := range entry.in {
			if utils.FuzzyMatch(entry.in[i], entry.out[i]) != nil {
				t.Errorf("Sorting error: %v vs %v", entry.in[i], entry.out[i])
			}
		}
	}
}

func TestMarkWithinQuota(t *testing.T) {
	s := getTestServer(".makrWithinQuota")
	c := &pb.Location{OverQuotaTime: time.Now().Unix()}
	s.markOverQuota(c, 0)

	if c.OverQuotaTime > 0 {
		t.Errorf("Quota has not been nulled out: %v", c)
	}
}
