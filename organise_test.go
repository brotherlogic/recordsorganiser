package main

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/brotherlogic/goserver"
	"github.com/brotherlogic/keystore/client"
	"golang.org/x/net/context"

	pbs "github.com/brotherlogic/discogssyncer/server"
	pbd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

type testBridgeFail struct{}

func (discogsBridge testBridgeFail) GetIP(name string) (string, int) {
	return "", -1
}

func (discogsBridge testBridgeFail) getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error) {
	return nil, errors.New("Built to fail")
}
func (discogsBridge testBridgeFail) getReleases(ctx context.Context, folders []int32) ([]*pbrc.Record, error) {
	return nil, errors.New("Built to fail")
}

func (discogsBridge testBridgeFail) getReleasesWithGoal(ctx context.Context, folders []int32) ([]*pbrc.Record, error) {
	return nil, errors.New("Built to fail")
}

func (discogsBridge testBridgeFail) getRelease(ID int32) (*pbd.Release, error) {
	return nil, errors.New("Built to fail")
}
func (discogsBridge testBridgeFail) moveToFolder(move *pbs.ReleaseMove) {
	//Do nothing
}

type testBridgePartialFail struct{}

func (discogsBridge testBridgePartialFail) GetIP(name string) (string, int) {
	return "", -1
}

func (discogsBridge testBridgePartialFail) getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error) {
	return nil, errors.New("Built to fail")
}
func (discogsBridge testBridgePartialFail) getReleases(ctx context.Context, olders []int32) ([]*pbrc.Record, error) {
	var result []*pbrc.Record

	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             1,
		MasterId:       10,
		Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 2,
	}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             2,
		Labels:         []*pbd.Label{&pbd.Label{Name: "SecondLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 1,
	}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             3,
		Labels:         []*pbd.Label{&pbd.Label{Name: "ThirdLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"CD"}}},
		FormatQuantity: 1,
	}})

	return result, nil
}
func (discogsBridge testBridgePartialFail) getReleaseWithGoals(ctx context.Context, olders []int32) ([]*pbrc.Record, error) {
	var result []*pbrc.Record

	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             1,
		MasterId:       10,
		Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 2,
	}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             2,
		Labels:         []*pbd.Label{&pbd.Label{Name: "SecondLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 1,
	}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             3,
		Labels:         []*pbd.Label{&pbd.Label{Name: "ThirdLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"CD"}}},
		FormatQuantity: 1,
	}})

	return result, nil
}

func (discogsBridge testBridgePartialFail) getRelease(ctx context.Context, ID int32) (*pbd.Release, error) {
	return nil, errors.New("Built to fail")
}
func (discogsBridge testBridgePartialFail) moveToFolder(move *pbs.ReleaseMove) {
	//Do nothing
}

type testBridgeCleverFail struct{}

func (discogsBridge testBridgeCleverFail) GetIP(name string) (string, int) {
	return "", -1
}

func (discogsBridge testBridgeCleverFail) getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error) {
	metadata := &pbrc.ReleaseMetadata{}
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
func (discogsBridge testBridgeCleverFail) getReleases(ctx context.Context, folders []int32) ([]*pbrc.Record, error) {
	for _, fold := range folders {
		if fold != 673768 {
			return nil, errors.New("Built to fail")
		}
	}
	var result []*pbrc.Record

	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             1,
		MasterId:       10,
		Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 2,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().Unix()}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             2,
		Labels:         []*pbd.Label{&pbd.Label{Name: "SecondLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 1,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().Unix() - 100}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             3,
		Labels:         []*pbd.Label{&pbd.Label{Name: "ThirdLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"CD"}}},
		FormatQuantity: 1,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().Unix() + 100}})

	return result, nil
}
func (discogsBridge testBridgeCleverFail) getReleasesWithGoal(ctx context.Context, folders []int32) ([]*pbrc.Record, error) {
	for _, fold := range folders {
		if fold != 673768 {
			return nil, errors.New("Built to fail")
		}
	}
	var result []*pbrc.Record

	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             1,
		Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 2,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().Unix()}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             2,
		Labels:         []*pbd.Label{&pbd.Label{Name: "SecondLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 1,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().Unix() - 100}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             3,
		Labels:         []*pbd.Label{&pbd.Label{Name: "ThirdLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"CD"}}},
		FormatQuantity: 1,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().Unix() + 100}})

	return result, nil
}

func (discogsBridge testBridgeCleverFail) getRelease(ID int32) (*pbd.Release, error) {
	return nil, errors.New("Built to fail")
}
func (discogsBridge testBridgeCleverFail) moveToFolder(move *pbs.ReleaseMove) {
	//Do nothing
}

type testBridge struct {
	widthMissing    bool
	failGetReleases bool
	failGetRecord   bool
}

type testBridgeMove struct {
	move bool
}

func (discogsBridge testBridgeMove) GetIP(name string) (string, int) {
	return "", -1
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

func (discogsBridge testBridgeMove) getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error) {
	metadata := &pbrc.ReleaseMetadata{GoalFolder: 25}
	switch instanceID {
	case 1:
		metadata.DateAdded = time.Now().Unix()
	case 2:
		metadata.DateAdded = time.Now().AddDate(0, -4, 0).Unix()
	case 3:
		metadata.DateAdded = time.Now().AddDate(0, -4, 0).Unix()
	}
	return &pbrc.Record{Release: &pbd.Release{InstanceId: 12}, Metadata: metadata}, nil
}

func (discogsBridge testBridge) getReleases(ctx context.Context, folders []int32) ([]*pbrc.Record, error) {
	if discogsBridge.failGetReleases {
		return []*pbrc.Record{}, fmt.Errorf("Built to fail")
	}

	if len(folders) == 1 && folders[0] == 812802 {
		return []*pbrc.Record{
			&pbrc.Record{
				Release: &pbd.Release{
					MasterId:       10,
					Id:             1,
					Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
					Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
					FormatQuantity: 2,
				},
				Metadata: &pbrc.ReleaseMetadata{GoalFolder: 25, Category: pbrc.ReleaseMetadata_GRADUATE}},
			&pbrc.Record{
				Release: &pbd.Release{
					Id:             1,
					Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
					Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
					FormatQuantity: 2,
				},
				Metadata: &pbrc.ReleaseMetadata{GoalFolder: 25}},
		}, nil
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

	return result, nil
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

func (discogsBridge testBridgeMove) getReleases(ctx context.Context, folders []int32) ([]*pbrc.Record, error) {
	var result []*pbrc.Record

	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             1,
		Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 2,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().Unix() + 100}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             2,
		Labels:         []*pbd.Label{&pbd.Label{Name: "SecondLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 1,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().AddDate(0, -4, 0).Unix()}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             3,
		Labels:         []*pbd.Label{&pbd.Label{Name: "FourthLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 1,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().AddDate(0, -4, 0).Unix()}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             4,
		Labels:         []*pbd.Label{&pbd.Label{Name: "ThirdLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"CD"}}},
		FormatQuantity: 1,
		Rating:         5,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().AddDate(0, -4, 0).Unix()}})

	return result, nil
}

func (discogsBridge testBridgeMove) getReleasesWithGoal(ctx context.Context, folders []int32) ([]*pbrc.Record, error) {
	var result []*pbrc.Record

	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             1,
		Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 2,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().Unix() + 100}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             2,
		Labels:         []*pbd.Label{&pbd.Label{Name: "SecondLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 1,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().AddDate(0, -4, 0).Unix()}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             3,
		Labels:         []*pbd.Label{&pbd.Label{Name: "FourthLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}},
		FormatQuantity: 1,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().AddDate(0, -4, 0).Unix()}})
	result = append(result, &pbrc.Record{Release: &pbd.Release{
		Id:             4,
		Labels:         []*pbd.Label{&pbd.Label{Name: "ThirdLabel"}},
		Formats:        []*pbd.Format{&pbd.Format{Descriptions: []string{"CD"}}},
		FormatQuantity: 1,
		Rating:         5,
	}, Metadata: &pbrc.ReleaseMetadata{DateAdded: time.Now().AddDate(0, -4, 0).Unix()}})

	return result, nil
}

func (discogsBridge testBridge) getRelease(ID int32) (*pbd.Release, error) {
	if ID < 3 {
		return &pbd.Release{Id: ID, Formats: []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}}, Labels: []*pbd.Label{&pbd.Label{Name: "SomethingElse"}}}, nil
	}
	return &pbd.Release{Id: ID, Formats: []*pbd.Format{&pbd.Format{Descriptions: []string{"CD"}}}, Labels: []*pbd.Label{&pbd.Label{Name: "Numero"}}}, nil
}

func (discogsBridge testBridgeMove) getRelease(ID int32) (*pbd.Release, error) {
	if ID < 4 {
		return &pbd.Release{Id: ID, Formats: []*pbd.Format{&pbd.Format{Descriptions: []string{"12"}}}, Labels: []*pbd.Label{&pbd.Label{Name: "SomethingElse"}}}, nil
	}
	return &pbd.Release{Id: ID, Formats: []*pbd.Format{&pbd.Format{Descriptions: []string{"CD"}}}, Labels: []*pbd.Label{&pbd.Label{Name: "Numero"}}}, nil
}

func (discogsBridge testBridge) moveToFolder(move *pbs.ReleaseMove) {
	//Do nothing
}

func (discogsBridge testBridgeMove) moveToFolder(move *pbs.ReleaseMove) {
	//Do nothing
}

func getTestServer(dir string) *Server {
	testServer := &Server{GoServer: &goserver.GoServer{}, bridge: testBridge{}, org: &pb.Organisation{}, gh: &testgh{}}
	testServer.Register = testServer
	testServer.GoServer.KSclient = *keystoreclient.GetTestClient(dir)
	testServer.SkipLog = true
	testServer.org.Extractors = append(testServer.org.Extractors, &pb.LabelExtractor{LabelId: 123, Extractor: "\\d\\d"})
	return testServer
}

func getTestServerWithMove(dir string) *Server {
	testServer := &Server{GoServer: &goserver.GoServer{}, bridge: testBridgeMove{}, org: &pb.Organisation{}}
	testServer.Register = testServer
	testServer.GoServer.KSclient = *keystoreclient.GetTestClient(dir)
	testServer.SkipLog = true
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
	testServer.bridge = testBridgeFail{}
	location := &pb.Location{
		Name:      "TestName",
		Slots:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_DATE_ADDED,
	}

	_, err := testServer.AddLocation(context.Background(), &pb.AddLocationRequest{Add: location})
	if err == nil {
		t.Errorf("Error on adding location: %v", err)
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
