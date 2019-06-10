package main

import (
	"log"
	"sort"
	"testing"

	pbgd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"

	"github.com/brotherlogic/goserver/utils"
	"golang.org/x/net/context"
)

var orderData = []struct {
	in  []*pbrc.Record
	out []*pbrc.Record
}{
	{
		// Later releases should be placed before earlier ones
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{Released: "2001"}},
			&pbrc.Record{Release: &pbgd.Release{Released: "2002"}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{Released: "2002"}},
			&pbrc.Record{Release: &pbgd.Release{Released: "2001"}},
		},
	},
	{
		// Later releases should be placed before earlier ones
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{Released: "2002"}},
			&pbrc.Record{Release: &pbgd.Release{Released: "2001"}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{Released: "2002"}},
			&pbrc.Record{Release: &pbgd.Release{Released: "2001"}},
		},
	},
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
	},
	{
		// NOT_KEEPER should come before KEEPER
		[]*pbrc.Record{
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_NOT_KEEPER}},
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_KEEPER}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_NOT_KEEPER}},
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_KEEPER}},
		},
	}}

func TestSaleOrdering(t *testing.T) {
	for _, entry := range orderData {
		sort.Sort(BySaleOrder(entry.in))
		for i := range entry.in {
			if !utils.FuzzyMatch(entry.in[i], entry.out[i]) {
				t.Errorf("Sorting error: %v vs %v", entry.in[i], entry.out[i])
			}
		}
	}
}

func TestSaleQuota(t *testing.T) {
	testLocation := &pb.Location{
		Name: "testing",
		Quota: &pb.Quota{
			NumOfSlots: 1,
		},
		ReleasesLocation: []*pb.ReleasePlacement{
			&pb.ReleasePlacement{},
			&pb.ReleasePlacement{},
		}}
	s := getTestServer(".testsalequota")
	s.processQuota(context.Background(), testLocation)
}

func TestFailRecordPull(t *testing.T) {
	testLocation := &pb.Location{
		Name: "testing",
		Quota: &pb.Quota{
			NumOfSlots: 1,
		},
		ReleasesLocation: []*pb.ReleasePlacement{
			&pb.ReleasePlacement{InstanceId: 1234},
			&pb.ReleasePlacement{},
		}}
	s := getTestServer(".testsalequota")
	s.bridge = &testBridge{failGetRecord: true}
	err := s.processQuota(context.Background(), testLocation)

	log.Printf("Boing %v", err)
	if err == nil {
		t.Errorf("Test Did not fail")
	}
}
