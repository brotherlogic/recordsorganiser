package main

import (
	"fmt"
	"io/ioutil"
	"testing"

	dpb "github.com/brotherlogic/godiscogs"
	rcpb "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
	"google.golang.org/protobuf/proto"
	"golang.org/x/net/context"
)

func loadTestRecord(iid int32) *rcpb.Record {
	data, err := ioutil.ReadFile(fmt.Sprintf("testdata/%v", iid))
	if err != nil {
		return nil
	}

	record := &rcpb.Record{}
	err = proto.Unmarshal(data, record)
	if err != nil {
		return nil
	}
	return record
}

func InitTestServer() *Server {
	server := InitServer()
	server.SkipIssue = true
	server.SkipLog = true

	return server
}

func TestCompareCollapse(t *testing.T) {
	server := InitTestServer()
	tests := []struct {
		r1 int32
		r2 int32
	}{
		{r1: 119991743, r2: 119992070},
	}

	for _, te := range tests {
		r1 := loadTestRecord(te.r1)
		r2 := loadTestRecord(te.r2)

		if r1 == nil || r2 == nil {
			t.Fatalf("Unable to load records")
		}
		cache := &pb.SortingCache{}
		appendCache(cache, r1)
		appendCache(cache, r2)

		server.labelMatch(context.Background(), r1, r2, cache)

		if server.IssueCount > 0 {
			t.Fatalf("Issue raised in label comparison: %v, %v", server.IssueCount, server.SkipIssue)
		}

	}
}

func TestBasicFunc(t *testing.T) {
	s := InitTestServer()
	records := []*rcpb.Record{
		{Release: &dpb.Release{
			InstanceId: 1234,
			Labels: []*dpb.Label{
				{Name: "Hudson"},
			},
		}, Metadata: &rcpb.ReleaseMetadata{
			RecordWidth: 123,
		}},
		{Release: &dpb.Release{
			InstanceId: 1235,
			Labels: []*dpb.Label{
				{Name: "Hudson"},
			},
		}, Metadata: &rcpb.ReleaseMetadata{
			RecordWidth: 234,
		}},
		{Release: &dpb.Release{
			InstanceId: 1236,
			Labels: []*dpb.Label{
				{Name: "Magic"},
			},
		}, Metadata: &rcpb.ReleaseMetadata{
			RecordWidth: 125,
		}},
	}

	nrecs, mapper := s.collapse(context.Background(), records, &pb.SortingCache{})

	if len(nrecs) != 2 {
		t.Errorf("Should be two records here: %v", nrecs)
	}

	nnrecs := expand(nrecs, mapper)

	if len(nnrecs) != 3 {
		t.Errorf("Should be three records here: %v", nnrecs)
	}

	if nnrecs[0].GetMetadata().GetRecordWidth() != 123 {
		t.Errorf("Should be 123: %v", nnrecs[0])
	}
}

func TestReverseFunc(t *testing.T) {
	s := InitTestServer()
	records := []*rcpb.Record{
		{Release: &dpb.Release{
			InstanceId: 1236,
			Labels: []*dpb.Label{
				{Name: "Magic"},
			},
		}, Metadata: &rcpb.ReleaseMetadata{
			RecordWidth: 125,
		}},
		{Release: &dpb.Release{
			InstanceId: 1234,
			Labels: []*dpb.Label{
				{Name: "Hudson"},
			},
		}, Metadata: &rcpb.ReleaseMetadata{
			RecordWidth: 123,
		}},
		{Release: &dpb.Release{
			InstanceId: 1235,
			Labels: []*dpb.Label{
				{Name: "Hudson"},
			},
		}, Metadata: &rcpb.ReleaseMetadata{
			RecordWidth: 234,
		}},
		{Release: &dpb.Release{
			InstanceId: 1236,
			Labels: []*dpb.Label{
				{Name: "Magic"},
			},
		}, Metadata: &rcpb.ReleaseMetadata{
			RecordWidth: 125,
		}},
	}

	nrecs, mapper := s.collapse(context.Background(), records, &pb.SortingCache{})

	if len(nrecs) != 3 {
		t.Errorf("Should be two records here: %v", nrecs)
	}

	nnrecs := expand(nrecs, mapper)

	if len(nnrecs) != 4 {
		t.Errorf("Should be three records here: %v", nnrecs)
	}

	if nnrecs[0].GetMetadata().GetRecordWidth() != 125 {
		t.Errorf("Should be 123: %v", nnrecs[0])
	}
}
