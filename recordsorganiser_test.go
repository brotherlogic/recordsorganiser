package main

import (
	"testing"
	"time"

	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
	"golang.org/x/net/context"
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

func TestMarkWithinQuota(t *testing.T) {
	s := getTestServer(".makrWithinQuota")
	c := &pb.Location{OverQuotaTime: time.Now().Unix()}
	s.markOverQuota(context.Background(), c)

	if c.OverQuotaTime > 0 {
		t.Errorf("Quota has not been nulled out: %v", c)
	}
}
