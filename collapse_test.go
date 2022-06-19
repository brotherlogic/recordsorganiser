package main

import (
	"testing"

	dpb "github.com/brotherlogic/godiscogs"
	rcpb "github.com/brotherlogic/recordcollection/proto"
)

func TestBasicFunc(t *testing.T) {
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

	nrecs, mapper := collapse(records)

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

	nrecs, mapper := collapse(records)

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
