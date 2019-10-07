package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/brotherlogic/goserver"
	"github.com/brotherlogic/keystore/client"
	"golang.org/x/net/context"

	pbd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

type testBridge struct {
	widthMissing    bool
	failGetReleases bool
	failGetRecord   bool
}

func (discogsBridge testBridge) GetIP(name string) (string, int) {
	return "", -1
}

func (discogsBridge testBridge) getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error) {
	if discogsBridge.failGetRecord {
		return nil, fmt.Errorf("Built to fail")
	}
	metadata := &pbrc.ReleaseMetadata{GoalFolder: 25, SpineWidth: 1}
	if discogsBridge.widthMissing {
		metadata.SpineWidth = 0
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

func (discogsBridge testBridge) getReleases(ctx context.Context, folders []int32) ([]int32, error) {
	if discogsBridge.failGetReleases {
		return []int32{}, fmt.Errorf("Built to fail")
	}

	if len(folders) == 1 && folders[0] == 812802 {
		return []int32{1, 2}, nil
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

	ids := []int32{}
	for _, r := range result {
		ids = append(ids, r.GetRelease().InstanceId)
	}

	return ids, nil
}

func (discogsBridge testBridge) getReleasesWithGoal(ctx context.Context, folders []int32) ([]*pbrc.Record, error) {
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

func (discogsBridge testBridge) getRelease(ID int32) (*pbd.Release, error) {
	if ID < 3 {
		return &pbd.Release{Id: ID, Formats: []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}}, Labels: []*pbd.Label{&pbd.Label{Name: "SomethingElse"}}}, nil
	}
	return &pbd.Release{Id: ID, Formats: []*pbd.Format{&pbd.Format{Descriptions: []string{"CD"}}}, Labels: []*pbd.Label{&pbd.Label{Name: "Numero"}}}, nil
}

func (discogsBridge testBridge) updateRecord(ctx context.Context, req *pbrc.UpdateRecordRequest) (*pbrc.UpdateRecordsResponse, error) {
	return &pbrc.UpdateRecordsResponse{}, nil
}

func getTestServer(dir string) *Server {
	testServer := &Server{GoServer: &goserver.GoServer{}, bridge: testBridge{}, org: &pb.Organisation{}, gh: &testgh{}}
	testServer.Register = testServer
	testServer.GoServer.KSclient = *keystoreclient.GetTestClient(dir)
	testServer.SkipLog = true
	testServer.org.Extractors = append(testServer.org.Extractors, &pb.LabelExtractor{LabelId: 123, Extractor: "\\d\\d"})
	testServer.scNeeded = make(map[string]int64)
	return testServer
}

func TestAddLocation(t *testing.T) {
	testServer := getTestServer(".testAddLocation")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}

	_, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Errorf("Error on adding location: %v", err)
	}
}

func TestAddLocationByDate(t *testing.T) {
	testServer := getTestServer(".testAddLocation")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_DATE_ADDED,
	}

	_, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Errorf("Error on adding location: %v", err)
	}
}

func TestAddLocationFail(t *testing.T) {
	testServer := getTestServer(".testAddLocation")
	testServer.bridge = testBridge{failGetReleases: true}
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_DATE_ADDED,
	}

	_, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err == nil {
		t.Errorf("Adding a location did not fail")
	}
}
func TestRaiseIssue(t *testing.T) {
	testServer := getTestServer(".testRaiseIssue")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}

	_, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Errorf("Error on adding location: %v", err)
	}

	testServer.checkQuota(context.Background())
}

func TestPassQuota(t *testing.T) {
	testServer := getTestServer(".testRaiseIssue")
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
		Quota: &pb.Quota{
			QuotaType: &pb.Quota_Slots{Slots: 20},
		},
	}

	_, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err != nil {
		t.Errorf("Error on adding location: %v", err)
	}

	err = testServer.reOrg(context.Background())
	if err != nil {
		t.Errorf("Error on reorg: %v", err)
	}
}
