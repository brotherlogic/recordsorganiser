package main

import (
	"testing"

	"golang.org/x/net/context"

	pb "github.com/brotherlogic/recordsorganiser/proto"
)

func TestBadReleaseGet(t *testing.T) {
	s := getTestServer(".testbadreleaseget")
	s.bridge = testBridge{failGetReleases: true}

	recs := s.getRecordsForFolder(context.Background(), &pb.Location{})

	if len(recs) != 0 {
		t.Errorf("Bad bridge retrieve did not fail quota pull")
	}
}

func TestBadRecordReleaseGet(t *testing.T) {
	s := getTestServer(".testbadreleaseget")
	s.bridge = testBridge{failGetRecord: true}

	recs := s.getRecordsForFolder(context.Background(), &pb.Location{})

	if len(recs) != 0 {
		t.Errorf("Bad bridge retrieve did not fail quota pull")
	}
}

func TestReleaseGet(t *testing.T) {
	s := getTestServer(".testbadreleaseget")
	s.bridge = testBridge{}

	recs := s.getRecordsForFolder(context.Background(), &pb.Location{FolderIds: []int32{25}})

	if len(recs) != 3 {
		t.Errorf("Not enough records returned: %v -> %v", recs, len(recs))
	}
}
