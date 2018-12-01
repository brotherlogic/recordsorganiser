package main

import (
	"testing"

	"golang.org/x/net/context"

	pb "github.com/brotherlogic/recordsorganiser/proto"
)

func TestLocate(t *testing.T) {
	testServer := getTestServer(".testLocate")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     1,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_DATE_ADDED,
		ReleasesLocation: []*pb.ReleasePlacement{
			&pb.ReleasePlacement{InstanceId: 1234, Index: 1, Slot: 1},
		},
	}
	testServer.org.Locations = append(testServer.org.Locations, location)

	f, err := testServer.Locate(context.Background(), &pb.LocateRequest{InstanceId: 1234})

	if err != nil {
		t.Fatalf("Error locating record: %v", err)
	}

	if f.FoundLocation.GetName() != "TestName" {
		t.Errorf("Error on spotted location: %v", f)
	}
}

func TestLocateFail(t *testing.T) {
	testServer := getTestServer(".testLocate")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     1,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_DATE_ADDED,
		ReleasesLocation: []*pb.ReleasePlacement{
			&pb.ReleasePlacement{InstanceId: 1234, Index: 1, Slot: 1},
		},
	}
	testServer.org.Locations = append(testServer.org.Locations, location)

	f, err := testServer.Locate(context.Background(), &pb.LocateRequest{InstanceId: 12345})

	if err == nil {
		t.Fatalf("Failed locate has not failed: %v", f)
	}
}

func TestGetLocation(t *testing.T) {
	testServer := getTestServer(".testAddLocation")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}

	_, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Fatalf("Unable to add location: %v", err)
	}

	resp, err := testServer.GetOrganisation(context.Background(), &pb.GetOrganisationRequest{ForceReorg: true, Locations: []*pb.Location{&pb.Location{Name: "TestName"}}})
	if err != nil {
		t.Fatalf("Unable to get organisation %v", err)
	}

	if len(resp.GetLocations()) != 1 {
		t.Errorf("Bad location response: %v", resp)
	}
}

func TestGetLocationReorg(t *testing.T) {
	testServer := getTestServer(".testAddLocation")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}

	_, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Fatalf("Unable to add location: %v", err)
	}

	resp, err := testServer.GetOrganisation(context.Background(), &pb.GetOrganisationRequest{ForceReorg: true, Locations: []*pb.Location{&pb.Location{Name: "TestName"}}})
	if err != nil {
		t.Fatalf("Unable to get organisation %v", err)
	}

	orgTime1 := resp.GetLocations()[0].LastReorg

	resp, err = testServer.GetOrganisation(context.Background(), &pb.GetOrganisationRequest{OrgReset: true, ForceReorg: true, Locations: []*pb.Location{&pb.Location{Name: "TestName"}}})
	if err != nil {
		t.Fatalf("Unable to get organisation %v", err)
	}

	if resp.GetLocations()[0].LastReorg == orgTime1 {
		t.Errorf("Reorg time not updated!: %v", orgTime1)
	}

}

func TestGetLocationOrgFail(t *testing.T) {
	testServer := getTestServer(".testAddLocation")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}

	_, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Fatalf("Unable to add location: %v", err)
	}

	testServer.bridge = testBridgeFail{}

	_, err = testServer.GetOrganisation(context.Background(), &pb.GetOrganisationRequest{ForceReorg: true, Locations: []*pb.Location{&pb.Location{Name: "TestName"}}})
	if err == nil {
		t.Fatalf("Failing bridge did not fail reorg %v", err)
	}
}

func TestUpdateLocation(t *testing.T) {
	testServer := getTestServer(".testUpdate")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
		Quota:     &pb.Quota{NumOfSlots: 2},
	}

	_, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Fatalf("Error in adding location: %v", err)
	}

	_, err = testServer.UpdateLocation(context.Background(), &pb.UpdateLocationRequest{Location: "TestName", Update: &pb.Location{Quota: &pb.Quota{NumOfSlots: 3}}})
	if err != nil {
		t.Fatalf("Error adding location: %v", err)
	}

	locs, err := testServer.GetOrganisation(context.Background(), &pb.GetOrganisationRequest{Locations: []*pb.Location{&pb.Location{Name: "TestName"}}})
	if err != nil {
		t.Fatalf("Can't get org")
	}

	if len(locs.GetLocations()) != 1 {
		t.Fatalf("Error getting org: %v", locs)
	}

	if locs.GetLocations()[0].GetQuota().GetNumOfSlots() != 3 {
		t.Errorf("Update failed: %v", locs)
	}
}

type testgh struct{}

func (t *testgh) alert(ctx context.Context, r *pb.Location) error {
	return nil
}

func TestGetOverQuota(t *testing.T) {
	testServer := getTestServer(".testQuota")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{25},
		Sort:      pb.Location_BY_LABEL_CATNO,
		Quota:     &pb.Quota{NumOfSlots: 2},
	}

	l, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Fatalf("Error in adding location: %v", err)
	}

	if len(l.GetNow().GetLocations()[0].GetReleasesLocation()) == 0 {
		t.Fatalf("No releases at the new location")
	}

	quota, err := testServer.GetQuota(context.Background(), &pb.QuotaRequest{FolderId: 25, IncludeRecords: true})

	if err != nil {
		t.Fatalf("Error getting quota: %v", err)
	}

	if !quota.GetOverQuota() {
		t.Errorf("Reported under quota?: %v", quota)
	}
}

func TestGetOverQuotaWithListeningPile(t *testing.T) {
	testServer := getTestServer(".testQuotaWithPile")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     1,
		FolderIds: []int32{25},
		Sort:      pb.Location_BY_LABEL_CATNO,
		Quota:     &pb.Quota{NumOfSlots: 4},
	}

	l, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Fatalf("Error in adding location: %v", err)
	}

	pile := &pb.Location{
		Name:      "Listening Pile",
		Slots:     1,
		FolderIds: []int32{812802},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}
	_, err = testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: pile})
	if err != nil {
		t.Fatalf("Error in adding location: %v", err)
	}

	if len(l.GetNow().GetLocations()[0].GetReleasesLocation()) == 0 {
		t.Fatalf("No releases at the new location")
	}

	quota, err := testServer.GetQuota(context.Background(), &pb.QuotaRequest{FolderId: 25, IncludeRecords: true})

	if err != nil {
		t.Fatalf("Error getting quota: %v", err)
	}

	if !quota.GetOverQuota() {
		t.Errorf("Reported under quota?: %v", quota)
	}
}

func TestGetUnderQuota(t *testing.T) {
	testServer := getTestServer(".testQuota")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
		Quota:     &pb.Quota{NumOfSlots: 4},
	}

	l, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Fatalf("Error in adding location: %v", err)
	}

	if len(l.GetNow().GetLocations()[0].GetReleasesLocation()) == 0 {
		t.Fatalf("No releases at the new location")
	}

	quota, err := testServer.GetQuota(context.Background(), &pb.QuotaRequest{FolderId: 10, IncludeRecords: true})

	if err != nil {
		t.Fatalf("Error getting quota: %v", err)
	}

	if quota.GetOverQuota() {
		t.Errorf("Reported over quota?: %v", quota)
	}
}

func TestGetUnderQuotaByName(t *testing.T) {
	testServer := getTestServer(".testQuota")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
		Quota:     &pb.Quota{NumOfSlots: 4},
	}

	l, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Fatalf("Error in adding location: %v", err)
	}

	if len(l.GetNow().GetLocations()[0].GetReleasesLocation()) == 0 {
		t.Fatalf("No releases at the new location")
	}

	quota, err := testServer.GetQuota(context.Background(), &pb.QuotaRequest{Name: "TestName", IncludeRecords: true})

	if err != nil {
		t.Fatalf("Error getting quota: %v", err)
	}

	if quota.GetOverQuota() {
		t.Errorf("Reported over quota?: %v", quota)
	}
}

func TestGetQuotaFail(t *testing.T) {
	testServer := getTestServer(".testQuota")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
		Quota:     &pb.Quota{NumOfSlots: 4},
	}

	l, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Fatalf("Error in adding location: %v", err)
	}

	if len(l.GetNow().GetLocations()[0].GetReleasesLocation()) == 0 {
		t.Fatalf("No releases at the new location")
	}

	_, err = testServer.GetQuota(context.Background(), &pb.QuotaRequest{FolderId: 20, IncludeRecords: true})

	if err == nil {
		t.Errorf("No errror on bad quota: %v", err)
	}
}

func TestAddExtractor(t *testing.T) {
	testServer := getTestServer(".addExtractor")
	_, err := testServer.AddExtractor(context.Background(), &pb.AddExtractorRequest{Extractor: &pb.LabelExtractor{LabelId: 1234, Extractor: "hello"}})
	if err != nil {
		t.Errorf("Error adding label")
	}
}

func TestGetOverQuotaNewSlots(t *testing.T) {
	testServer := getTestServer(".testQuota")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{25},
		Sort:      pb.Location_BY_LABEL_CATNO,
		Quota:     &pb.Quota{QuotaType: &pb.Quota_Slots{Slots: 2}},
	}

	l, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Fatalf("Error in adding location: %v", err)
	}

	if len(l.GetNow().GetLocations()[0].GetReleasesLocation()) == 0 {
		t.Fatalf("No releases at the new location")
	}

	quota, err := testServer.GetQuota(context.Background(), &pb.QuotaRequest{FolderId: 25, IncludeRecords: true})

	if err != nil {
		t.Fatalf("Error getting quota: %v", err)
	}

	if !quota.GetOverQuota() {
		t.Errorf("Reported under quota?: %v", quota)
	}
}

func TestGetOverQuotaNewWidthWithNoWidth(t *testing.T) {
	testServer := getTestServer(".testQuota")
	testServer.bridge = &testBridge{widthMissing: true}
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{25},
		Sort:      pb.Location_BY_LABEL_CATNO,
		Quota:     &pb.Quota{QuotaType: &pb.Quota_Width{Width: 2}},
	}

	l, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Fatalf("Error in adding location: %v", err)
	}

	if len(l.GetNow().GetLocations()[0].GetReleasesLocation()) == 0 {
		t.Fatalf("No releases at the new location")
	}

	recs, err := testServer.GetQuota(context.Background(), &pb.QuotaRequest{FolderId: 25, IncludeRecords: true})

	if err == nil {
		t.Fatalf("Should have Error getting quota: %v", recs)
	}
}

func TestGetOverQuotaNewWidth(t *testing.T) {
	testServer := getTestServer(".testQuota")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{25},
		Sort:      pb.Location_BY_LABEL_CATNO,
		Quota:     &pb.Quota{QuotaType: &pb.Quota_Width{Width: 2}},
	}

	l, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Fatalf("Error in adding location: %v", err)
	}

	if len(l.GetNow().GetLocations()[0].GetReleasesLocation()) == 0 {
		t.Fatalf("No releases at the new location")
	}

	quota, err := testServer.GetQuota(context.Background(), &pb.QuotaRequest{FolderId: 25, IncludeRecords: true})

	if err != nil {
		t.Fatalf("Error getting quota: %v", err)
	}

	if !quota.GetOverQuota() {
		t.Errorf("Reported under quota?: %v", quota)
	}
}

func TestGetOverQuotaWithNoQuota(t *testing.T) {
	testServer := getTestServer(".testQuota")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}

	l, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Fatalf("Error in adding location: %v", err)
	}

	if len(l.GetNow().GetLocations()[0].GetReleasesLocation()) == 0 {
		t.Fatalf("No releases at the new location")
	}

	quota, err := testServer.GetQuota(context.Background(), &pb.QuotaRequest{FolderId: 10, IncludeRecords: true})

	if err == nil {
		t.Fatalf("Should have errored: %v", quota)
	}
}
