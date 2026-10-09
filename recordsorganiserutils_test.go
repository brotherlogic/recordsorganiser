package main

import (
	"log"
	"testing"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbdg "github.com/brotherlogic/discogs/proto"
	pbgr "github.com/brotherlogic/gramophile/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
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

func TestProcessSlotQuota_NoSaleAndNoAlert(t *testing.T) {
	testLocation := &pb.Location{
		Name:      "test_slot_quota_location",
		FolderIds: []int32{0},
		Quota: &pb.Quota{
			QuotaType: &pb.Quota_Slots{Slots: 1},
		},
		ReleasesLocation: []*pb.ReleasePlacement{
			&pb.ReleasePlacement{InstanceId: 1, Slot: 1},
			&pb.ReleasePlacement{InstanceId: 2, Slot: 2},
		},
	}
	s := getTestServer(".testprocessslotquota_nosaleandnoalert")
	err := s.processSlotQuota(context.Background(), testLocation)
	if err != nil {
		t.Fatalf("unexpected error running processSlotQuota: %v", err)
	}

	foundSlotsVal := testutil.ToFloat64(foundSlots.With(prometheus.Labels{"org": testLocation.GetName()}))
	if foundSlotsVal != 2 {
		t.Errorf("expected foundSlots gauge to be 2, got %v", foundSlotsVal)
	}

	spillVal := testutil.ToFloat64(spill.With(prometheus.Labels{"location": testLocation.GetName()}))
	if spillVal != 1 {
		t.Errorf("expected spill gauge to be 1, got %v", spillVal)
	}

	tb := s.bridge.(*testBridge)
	if len(tb.getUpdates()) != 0 {
		t.Errorf("expected zero calls to updateRecord when slots exceed quota, got %d", len(tb.getUpdates()))
	}

	if s.IssueCount != 0 {
		t.Errorf("expected zero alerts/issues raised, got %d", s.IssueCount)
	}
}

func TestProcessWidthQuota_NoSale(t *testing.T) {
	testLocation := &pb.Location{
		Name:  "test_width_quota_location",
		Slots: 1,
		Quota: &pb.Quota{
			TotalWidth: 5,
		},
		ReleasesLocation: []*pb.ReleasePlacement{
			&pb.ReleasePlacement{InstanceId: 1, Slot: 0},
			&pb.ReleasePlacement{InstanceId: 2, Slot: 0},
		},
	}
	s := getTestServer(".testprocesswidthquota_nosale")
	s.bridge = &testBridge{recordWidth: 10}
	err := s.processWidthQuota(context.Background(), testLocation)
	if err != nil {
		t.Fatalf("unexpected error running processWidthQuota: %v", err)
	}

	tb := s.bridge.(*testBridge)
	if len(tb.getUpdates()) != 0 {
		t.Errorf("expected zero calls to updateRecord when slot width exceeds quota, got %d", len(tb.getUpdates()))
	}
}

func TestDarkSaleCandidate_Success(t *testing.T) {
	s := getTestServer(".testdarksalecandidate_success")
	tb := &testBridge{
		candidateResp: &pbgr.RecordResponse{
			Record: &pbgr.Record{
				Release: &pbdg.Release{
					InstanceId: 12345,
					Title:      "Dark Launch Test Release",
				},
				PackageScore: 95,
				MedianPrice: &pbdg.Price{
					Currency: "USD",
					Value:    2500,
				},
			},
		},
	}
	s.bridge = tb

	initSuccessCount := testutil.ToFloat64(darkSaleCandidates.With(prometheus.Labels{"location": "12 Inches"}))

	loc := &pb.Location{Name: "12 Inches"}
	s.evaluateDarkSaleCandidate(context.Background(), loc)

	if len(tb.saleCandidateCalls) != 1 || tb.saleCandidateCalls[0] != "12 Inches" {
		t.Errorf("expected 1 call with '12 Inches', got %v", tb.saleCandidateCalls)
	}

	newSuccessCount := testutil.ToFloat64(darkSaleCandidates.With(prometheus.Labels{"location": "12 Inches"}))
	if newSuccessCount != initSuccessCount+1 {
		t.Errorf("expected darkSaleCandidates count to increase by 1, got from %v to %v", initSuccessCount, newSuccessCount)
	}
}

func TestDarkSaleCandidate_NotFound(t *testing.T) {
	s := getTestServer(".testdarksalecandidate_notfound")
	tb := &testBridge{
		candidateErr: status.Errorf(codes.NotFound, "no sale candidate found"),
	}
	s.bridge = tb

	initErrCount := testutil.ToFloat64(darkSaleCandidateErrors.With(prometheus.Labels{"location": "12 Inches"}))

	loc := &pb.Location{Name: "12 Inches"}
	s.evaluateDarkSaleCandidate(context.Background(), loc)

	if len(tb.saleCandidateCalls) != 1 || tb.saleCandidateCalls[0] != "12 Inches" {
		t.Errorf("expected 1 call with '12 Inches', got %v", tb.saleCandidateCalls)
	}

	newErrCount := testutil.ToFloat64(darkSaleCandidateErrors.With(prometheus.Labels{"location": "12 Inches"}))
	if newErrCount != initErrCount {
		t.Errorf("expected darkSaleCandidateErrors count to remain %v on NotFound, got %v", initErrCount, newErrCount)
	}
}

func TestDarkSaleCandidate_RPCFailure(t *testing.T) {
	s := getTestServer(".testdarksalecandidate_rpcfailure")
	tb := &testBridge{
		candidateErr: status.Errorf(codes.Unavailable, "service unavailable"),
	}
	s.bridge = tb

	initErrCount := testutil.ToFloat64(darkSaleCandidateErrors.With(prometheus.Labels{"location": "12 Inches"}))

	loc := &pb.Location{Name: "12 Inches"}
	s.evaluateDarkSaleCandidate(context.Background(), loc)

	if len(tb.saleCandidateCalls) != 1 || tb.saleCandidateCalls[0] != "12 Inches" {
		t.Errorf("expected 1 call with '12 Inches', got %v", tb.saleCandidateCalls)
	}

	newErrCount := testutil.ToFloat64(darkSaleCandidateErrors.With(prometheus.Labels{"location": "12 Inches"}))
	if newErrCount != initErrCount+1 {
		t.Errorf("expected darkSaleCandidateErrors count to increase by 1 on RPC error, got from %v to %v", initErrCount, newErrCount)
	}
}

func TestDarkSaleCandidate_UnmappedLocation(t *testing.T) {
	s := getTestServer(".testdarksalecandidate_unmapped")
	tb := &testBridge{}
	s.bridge = tb

	loc := &pb.Location{Name: "Unmapped Location"}
	s.evaluateDarkSaleCandidate(context.Background(), loc)

	if len(tb.saleCandidateCalls) != 0 {
		t.Errorf("expected 0 calls for unmapped location, got %v", tb.saleCandidateCalls)
	}
}


