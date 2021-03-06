package main

import (
	"log"
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
