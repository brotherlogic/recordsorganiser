package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/brotherlogic/goserver"
	keystoreclient "github.com/brotherlogic/keystore/client"
	"golang.org/x/net/context"

	pbd "github.com/brotherlogic/godiscogs/proto"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

type testBridge struct {
	sync.Mutex
	widthMissing    bool
	failGetReleases bool
	failGetRecord   bool
	recordWidth     float32
	updates         []*pbrc.UpdateRecordRequest
}

func (discogsBridge *testBridge) GetIP(name string) (string, int) {
	return "", -1
}

func (discogsBridge *testBridge) getRecord(ctx context.Context, instanceID int64) (*pbrc.Record, error) {
	if discogsBridge.failGetRecord {
		return nil, fmt.Errorf("Built to fail")
	}
	metadata := &pbrc.ReleaseMetadata{GoalFolder: 25, SpineWidth: 1}
	if discogsBridge.widthMissing {
		metadata.SpineWidth = 0
	}
	if discogsBridge.recordWidth > 0 {
		metadata.RecordWidth = discogsBridge.recordWidth
	}
	switch instanceID {
	case 1:
		metadata.DateAdded = time.Now().Unix()
	case 2:
		metadata.DateAdded = time.Now().Unix() - 100
	case 3:
		metadata.DateAdded = time.Now().Unix() + 100
	}
	return &pbrc.Record{Release: &pbd.Release{InstanceId: 12}, Metadata: metadata}, nil
}

func (discogsBridge *testBridge) getReleases(ctx context.Context, folders []int32) ([]int64, error) {
	if discogsBridge.failGetReleases {
		return []int64{}, fmt.Errorf("Built to fail")
	}

	if len(folders) == 1 && folders[0] == 812802 {
		return []int64{1, 2}, nil
		/*		return []*pbrc.Record{
				&pbrc.Record{
					Release: &pbd.Release{
						MasterId:       10,
						Id:             1,
						Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
						Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
						FormatQuantity: 2,
					},
					Metadata: &pbrc.ReleaseMetadata{GoalFolder: 25, Category: pbrc.ReleaseMetadata_ASSESS_FOR_SALE}},
				&pbrc.Record{
					Release: &pbd.Release{
						Id:             1,
						Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
						Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
						FormatQuantity: 2,
					},
					Metadata: &pbrc.ReleaseMetadata{GoalFolder: 25}},
			}, nil */
	}

	var result []*pbrc.Record

	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             1,
		MasterId:       10,
		Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 2,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().AddDate(0, -4, 0).Unix(), SpineWidth: 1, Category: pbrc.ReleaseMetadata_GRADUATE}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             2,
		Labels:         []*pbd.Label{&pbd.Label{Name: "SecondLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 1,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().AddDate(0, -4, 0).Unix(), SpineWidth: 1}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             3,
		Labels:         []*pbd.Label{&pbd.Label{Name: "ThirdLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"CD"}}},
		FormatQuantity: 1,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().AddDate(0, -4, 0).Unix(), SpineWidth: 1}})

	if discogsBridge.widthMissing {
		for _, r := range result {
			r.Metadata.SpineWidth = 0
		}
	}

	ids := []int64{}
	for _, r := range result {
		ids = append(ids, r.GetRelease().InstanceId)
	}

	return ids, nil
}

func (discogsBridge *testBridge) getReleasesWithGoal(ctx context.Context, folders []int32) ([]*pbrc.Record, error) {
	if discogsBridge.failGetReleases {
		return []*pbrc.Record{}, fmt.Errorf("Built to fail")
	}

	include25 := false
	for _, v := range folders {
		if v == 25 {
			include25 = true
		}
	}

	result := []*pbrc.Record{}

	if include25 {
		result = append(result, []*pbrc.Record{
			&pbrc.Record{
				Release: &pbd.Release{
					Id:             1,
					MasterId:       10,
					Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
					Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
					FormatQuantity: 2,
				},
				Metadata: &pbrc.ReleaseMetadata{GoalFolder: 25, SpineWidth: 1, Category: pbrc.ReleaseMetadata_GRADUATE}},
			&pbrc.Record{
				Release: &pbd.Release{
					Id:             1,
					MasterId:       10,
					Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
					Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
					FormatQuantity: 2,
				},
				Metadata: &pbrc.ReleaseMetadata{GoalFolder: 25, SpineWidth: 1, Category: pbrc.ReleaseMetadata_GRADUATE}},
		}...)
	}

	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             1,
		Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 2,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().AddDate(0, -4, 0).Unix(), GoalFolder: 25, SpineWidth: 1, Category: pbrc.ReleaseMetadata_GRADUATE}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             2,
		Labels:         []*pbd.Label{&pbd.Label{Name: "SecondLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 1,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().AddDate(0, -4, 0).Unix(), GoalFolder: 25, SpineWidth: 1, Category: pbrc.ReleaseMetadata_GRADUATE}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             3,
		Labels:         []*pbd.Label{&pbd.Label{Name: "ThirdLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"CD"}}},
		FormatQuantity: 1,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().AddDate(0, -4, 0).Unix(), GoalFolder: 25, SpineWidth: 1, Category: pbrc.ReleaseMetadata_GRADUATE}})

	if discogsBridge.widthMissing {
		for _, r := range result {
			r.Metadata.SpineWidth = 0
		}
	}

	return result, nil
}

func (discogsBridge *testBridge) getRelease(ID int32) (*pbd.Release, error) {
	if ID < 3 {
		return &pbd.Release{Id: ID, Formats: []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}}, Labels: []*pbd.Label{&pbd.Label{Name: "SomethingElse"}}}, nil
	}
	return &pbd.Release{Id: ID, Formats: []*pbd.Format{&pbd.Format{Descriptions: []string{"CD"}}}, Labels: []*pbd.Label{&pbd.Label{Name: "Numero"}}}, nil
}

func (discogsBridge *testBridge) updateRecord(ctx context.Context, req *pbrc.UpdateRecordRequest) (*pbrc.UpdateRecordsResponse, error) {
	discogsBridge.Lock()
	defer discogsBridge.Unlock()
	discogsBridge.updates = append(discogsBridge.updates, req)
	return &pbrc.UpdateRecordsResponse{}, nil
}

func (discogsBridge *testBridge) getUpdates() []*pbrc.UpdateRecordRequest {
	discogsBridge.Lock()
	defer discogsBridge.Unlock()
	copied := make([]*pbrc.UpdateRecordRequest, len(discogsBridge.updates))
	copy(copied, discogsBridge.updates)
	return copied
}

func (discogsBridge *testBridge) resetUpdates() {
	discogsBridge.Lock()
	defer discogsBridge.Unlock()
	discogsBridge.updates = nil
}

func getTestServer(dir string) *Server {
	testServer := &Server{GoServer: &goserver.GoServer{}, bridge: &testBridge{}}
	testServer.Register = testServer
	testServer.GoServer.KSclient = *keystoreclient.GetTestClient(dir)
	testServer.SkipLog = true
	testServer.SkipIssue = true
	org := &pb.Organisation{}
	org.Extractors = append(org.Extractors, &pb.LabelExtractor{LabelId: 123, Extractor: "\\d\\d"})
	testServer.GoServer.KSclient.Save(context.Background(), KEY, org)
	return testServer
}

func TestTestBridgeUpdateRecord(t *testing.T) {
	tb := &testBridge{}
	ctx := context.Background()

	req := &pbrc.UpdateRecordRequest{
		Reason: "Test Reason",
		Update: &pbrc.Record{
			Release: &pbd.Release{InstanceId: 12345},
		},
	}

	res, err := tb.updateRecord(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error from updateRecord: %v", err)
	}
	if res == nil {
		t.Fatalf("expected non-nil response from updateRecord")
	}

	updates := tb.getUpdates()
	if len(updates) != 1 {
		t.Fatalf("expected 1 recorded update, got %d", len(updates))
	}
	if updates[0].GetReason() != "Test Reason" {
		t.Errorf("unexpected recorded update reason: got %v, want %v", updates[0].GetReason(), "Test Reason")
	}

	tb.resetUpdates()
	if len(tb.getUpdates()) != 0 {
		t.Errorf("expected 0 recorded updates after reset, got %d", len(tb.getUpdates()))
	}
}
