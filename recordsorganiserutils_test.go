package main

import (
	"log"
	"testing"

	"golang.org/x/net/context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	pb "github.com/brotherlogic/recordsorganiser/proto"
)

func TestBadReleaseGet(t *testing.T) {
	s := getTestServer(".testbadreleaseget")
	s.bridge = &testBridge{failGetReleases: true}

	recs := s.getRecordsForFolder(context.Background(), &pb.Location{})

	if len(recs) != 0 {
		t.Errorf("Bad bridge retrieve did not fail quota pull")
	}
}

func TestBadRecordReleaseGet(t *testing.T) {
	s := getTestServer(".testbadreleaseget")
	s.bridge = &testBridge{failGetRecord: true}

	recs := s.getRecordsForFolder(context.Background(), &pb.Location{})

	if len(recs) != 0 {
		t.Errorf("Bad bridge retrieve did not fail quota pull")
	}
}

func TestReleaseGet(t *testing.T) {
	s := getTestServer(".testbadreleaseget")
	s.bridge = &testBridge{}

	recs := s.getRecordsForFolder(context.Background(), &pb.Location{FolderIds: []int32{25}})

	if len(recs) != 3 {
		t.Errorf("Not enough records returned: %v -> %v", recs, len(recs))
	}
}

func TestSaleQuota(t *testing.T) {
	testLocation := &pb.Location{
		Name:      "testing",
		FolderIds: []int32{0},
		Quota: &pb.Quota{
			NumOfSlots: 1,
		},
		ReleasesLocation: []*pb.ReleasePlacement{
			&pb.ReleasePlacement{InstanceId: 1},
			&pb.ReleasePlacement{InstanceId: 2},
		},
	}
	s := getTestServer(".testsalequota")
	err := s.processQuota(context.Background(), testLocation)
	if err != nil {
		t.Fatalf("unexpected error running processQuota: %v", err)
	}

	tb := s.bridge.(*testBridge)
	if len(tb.getUpdates()) != 0 {
		t.Errorf("expected zero calls to updateRecord when over quota, got %d", len(tb.getUpdates()))
	}
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

func TestProcessAbsoluteWidthQuota_NoSale(t *testing.T) {
	testLocation := &pb.Location{
		Name: "test_absolute_width_location",
		Quota: &pb.Quota{
			QuotaType: &pb.Quota_AbsoluteWidth{AbsoluteWidth: 10},
		},
		ReleasesLocation: []*pb.ReleasePlacement{
			&pb.ReleasePlacement{InstanceId: 1, DeterminedWidth: 8},
			&pb.ReleasePlacement{InstanceId: 2, DeterminedWidth: 8},
		},
	}
	s := getTestServer(".testprocessabsolutewidthquota_nosale")
	err := s.processAbsoluteWidthQuota(context.Background(), testLocation)
	if err != nil {
		t.Fatalf("unexpected error running processAbsoluteWidthQuota: %v", err)
	}

	gaugeVal := testutil.ToFloat64(gwidth.With(prometheus.Labels{"location": testLocation.GetName()}))
	if gaugeVal != 10 {
		t.Errorf("expected gwidth gauge to be 10, got %v", gaugeVal)
	}

	tb := s.bridge.(*testBridge)
	if len(tb.getUpdates()) != 0 {
		t.Errorf("expected zero calls to updateRecord when total width exceeds absolute width quota, got %d", len(tb.getUpdates()))
	}
}
