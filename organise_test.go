package main

import (
	"log"
	"os"
	"testing"
	"time"

	"golang.org/x/net/context"

	pbs "github.com/brotherlogic/discogssyncer/server"
	pbd "github.com/brotherlogic/godiscogs"
	pb "github.com/brotherlogic/recordsorganiser/proto"
	"github.com/golang/protobuf/proto"
)

type testBridge struct{}

func (discogsBridge testBridge) getMetadata(rel *pbd.Release) *pbs.ReleaseMetadata {
	metadata := &pbs.ReleaseMetadata{}
	switch rel.Id {
	case 1:
		metadata.DateAdded = time.Now().Unix()
	case 2:
		metadata.DateAdded = time.Now().Unix() - 100
	case 3:
		metadata.DateAdded = time.Now().Unix() + 100
	}
	return metadata
}

func (discogsBridge testBridge) getReleases(folders []int32) []*pbd.Release {
	var result []*pbd.Release

	result = append(result, &pbd.Release{
		Id:             1,
		Labels:         []*pbd.Label{&pbd.Label{Name: "FirstLabel"}},
		FormatQuantity: 2,
	})
	result = append(result, &pbd.Release{
		Id:             2,
		Labels:         []*pbd.Label{&pbd.Label{Name: "SecondLabel"}},
		FormatQuantity: 1,
	})
	result = append(result, &pbd.Release{
		Id:             3,
		Labels:         []*pbd.Label{&pbd.Label{Name: "ThirdLabel"}},
		FormatQuantity: 1,
	})

	return result
}

func (discogsBridge testBridge) getRelease(ID int32) *pbd.Release {
	return &pbd.Release{Id: ID}
}

func TestGetReleaseLocation(t *testing.T) {
	testServer := &Server{saveLocation: ".testgetlocation", bridge: testBridge{}, org: &pb.Organisation{}}
	location := &pb.Location{
		Name:      "TestName",
		Units:     1,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}
	testServer.AddLocation(context.Background(), location)

	relLocation, err := testServer.Locate(context.Background(), &pbd.Release{Id: 2})

	if err != nil {
		t.Errorf("GetReleaseLocation has failed: %v", err)
	}

	if relLocation == nil {
		t.Errorf("Err: %v", relLocation)
	}

	if relLocation.Slot != 1 {
		t.Errorf("Slot has come back wrong: %v", relLocation)
	}

	if relLocation.Before == nil || relLocation.Before.Id != 1 {
		t.Errorf("Release before location has come back wrong: %v", relLocation)
	}
	if relLocation.After == nil || relLocation.After.Id != 3 {
		t.Errorf("Release after location has come back wrong: %v", relLocation)
	}

	relLocation, _ = testServer.Locate(context.Background(), &pbd.Release{Id: 1})
	if relLocation.Before != nil {
		t.Errorf("Release location has come back wrong: %v", relLocation)
	}
}

func TestListLocations(t *testing.T) {
	testServer := &Server{saveLocation: ".testgetorgs", bridge: testBridge{}, org: &pb.Organisation{}}
	location := &pb.Location{
		Name:      "TestName",
		Units:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}

	testServer.AddLocation(context.Background(), location)

	org, err := testServer.GetOrganisation(context.Background(), &pb.Empty{})

	if err != nil {
		t.Errorf("Error retrieving current organisation")
	}

	if len(org.Locations) != 1 {
		t.Errorf("Too Many Locations: %v", org)
	}
	if org.Locations[0].Name != "TestName" {
		t.Errorf("Location name is incorrect: %v", org.Locations[0])
	}
}

func clean(s *Server) {
	os.RemoveAll(s.saveLocation)
}

func TestDiffFails(t *testing.T) {
	testServer := &Server{saveLocation: ".testdiff", bridge: testBridge{}, org: &pb.Organisation{}}
	clean(testServer)
	location := &pb.Location{
		Name:      "TestName",
		Units:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}

	testServer.AddLocation(context.Background(), location)
	timestamps, err := testServer.GetOrganisations(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Error gettting orgs %v", err)
	}

	if len(timestamps.GetOrganisations()) != 1 {
		t.Errorf("Too many organisations present: %v", len(timestamps.GetOrganisations()))
	}

	diff1, err := testServer.Diff(context.Background(), &pb.DiffRequest{StartTimestamp: 1234, EndTimestamp: timestamps.Organisations[0].Timestamp})
	if err == nil {
		t.Errorf("Failure to pull start timestamp bad: %v", diff1)
	}
	diff2, err := testServer.Diff(context.Background(), &pb.DiffRequest{EndTimestamp: 1234, StartTimestamp: timestamps.Organisations[0].Timestamp})
	if err == nil {
		t.Errorf("Failure to pull start timestamp bad: %v", diff2)
	}
}

func TestDiffFailsNamedLocation(t *testing.T) {
	testServer := &Server{saveLocation: ".testdiff", bridge: testBridge{}, org: &pb.Organisation{}}
	clean(testServer)
	location := &pb.Location{
		Name:      "TestName",
		Units:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}

	testServer.AddLocation(context.Background(), location)

	locationUpdate := &pb.Location{
		Sort: pb.Location_BY_DATE_ADDED,
		Name: "TestName",
	}
	//Wait 2 seconds to let the timestamps change
	time.Sleep(time.Second * 2)
	testServer.UpdateLocation(context.Background(), locationUpdate)

	timestamps, err := testServer.GetOrganisations(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Error gettting orgs %v", err)
	}

	if len(timestamps.GetOrganisations()) != 2 {
		t.Errorf("Too many organisations present: %v", len(timestamps.GetOrganisations()))
	}

	diffRequest := &pb.DiffRequest{
		StartTimestamp: timestamps.Organisations[0].Timestamp,
		EndTimestamp:   timestamps.Organisations[1].Timestamp,
		LocationName:   "TestNameBad",
		Slot:           1,
	}
	_, err = testServer.Diff(context.Background(), diffRequest)
	if err == nil {
		t.Errorf("No Error running diff with bad name %v", err)
	}
}

func TestDiff(t *testing.T) {
	testServer := &Server{saveLocation: ".testdiff", bridge: testBridge{}, org: &pb.Organisation{}}
	clean(testServer)
	location := &pb.Location{
		Name:      "TestName",
		Units:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}

	testServer.AddLocation(context.Background(), location)
	locationUpdate := &pb.Location{
		Sort: pb.Location_BY_DATE_ADDED,
		Name: "TestName",
	}
	//Wait 2 seconds to let the timestamps change
	time.Sleep(time.Second * 2)
	testServer.UpdateLocation(context.Background(), locationUpdate)

	timestamps, err := testServer.GetOrganisations(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Error gettting orgs %v", err)
	}

	if len(timestamps.GetOrganisations()) != 2 {
		t.Errorf("Too many organisations present: %v", len(timestamps.GetOrganisations()))
	}

	diffRequest := &pb.DiffRequest{
		StartTimestamp: timestamps.Organisations[0].Timestamp,
		EndTimestamp:   timestamps.Organisations[1].Timestamp,
		LocationName:   "TestName",
		Slot:           1,
	}
	moves, err := testServer.Diff(context.Background(), diffRequest)
	if err != nil {
		t.Errorf("Error running diff %v", err)
	}
	if len(moves.Moves) != 2 {
		t.Errorf("Moves are wrong on diff: %v", moves)
	}
}

func TestGetOrganisations(t *testing.T) {
	testServer := &Server{saveLocation: ".testgetorgs", bridge: testBridge{}, org: &pb.Organisation{}}
	clean(testServer)
	testServer.save()

	//Sleep for 1.5 seconds to bump the timestamp
	time.Sleep(time.Millisecond * 1500)
	testServer.save()

	organisations, err := testServer.GetOrganisations(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Get Organisations returned an errors: %v", err)
	}
	if len(organisations.Organisations) != 2 {
		t.Errorf("Organisations has returned wrong: %v", organisations)
	}

	if organisations.Organisations[0].Timestamp <= 0 {
		t.Errorf("Timestamp has come back as wrong: %v", organisations.Organisations[0])
	}

	if organisations.Organisations[0].Timestamp == organisations.Organisations[1].Timestamp {
		t.Errorf("Timestamps are equal between two saves: %v", organisations.Organisations)
	}
}

func TestCompareMoves(t *testing.T) {
	start := []*pb.ReleasePlacement{
		&pb.ReleasePlacement{ReleaseId: 1, Index: 1, Slot: 1},
		&pb.ReleasePlacement{ReleaseId: 2, Index: 2, Slot: 1},
		&pb.ReleasePlacement{ReleaseId: 3, Index: 3, Slot: 1},
	}
	end := []*pb.ReleasePlacement{
		&pb.ReleasePlacement{ReleaseId: 1, Index: 2, Slot: 1},
		&pb.ReleasePlacement{ReleaseId: 2, Index: 1, Slot: 1},
		&pb.ReleasePlacement{ReleaseId: 4, Index: 3, Slot: 1},
	}
	expectedMoves := []*pb.LocationMove{
		&pb.LocationMove{Old: &pb.ReleasePlacement{ReleaseId: 3, Index: 3, BeforeReleaseId: 2}},
		&pb.LocationMove{New: &pb.ReleasePlacement{ReleaseId: 4, Index: 3, BeforeReleaseId: 2}},
		&pb.LocationMove{Old: &pb.ReleasePlacement{ReleaseId: 1, Index: 1, AfterReleaseId: 2}, New: &pb.ReleasePlacement{ReleaseId: 1, Index: 2, BeforeReleaseId: 2, AfterReleaseId: 4}},
	}

	moves := getMoves(start, end, 1)
	if len(moves) != 3 {
		t.Errorf("Not enough moves: %v", moves)
	}
	for i := range moves {
		if !proto.Equal(moves[i], expectedMoves[i]) {
			t.Errorf("Bad move : %v (expected %v)", moves[i], expectedMoves[i])
		}
	}
}

func TestGetLocation(t *testing.T) {
	testServer := &Server{saveLocation: ".testoutget", bridge: testBridge{}, org: &pb.Organisation{}}

	location := &pb.Location{
		Name:      "TestName",
		Units:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}

	testServer.AddLocation(context.Background(), location)

	testServer2 := &Server{saveLocation: ".testoutget", bridge: testBridge{}, org: &pb.Organisation{}}
	testServer2.org = loadLatest(".testoutget")
	retrLocation := &pb.Location{Name: "TestName"}
	retr, err := testServer2.GetLocation(context.Background(), retrLocation)
	if err != nil {
		t.Errorf("Error on getting location: %v", err)
	}

	if retr.Name != "TestName" {
		t.Errorf("Location name is wrong: %v", retrLocation)
	}

	if len(retr.FolderIds) == 0 || retr.FolderIds[0] != 10 {
		t.Errorf("Folder Id has come back wrong: %v", retrLocation)
	}
}

func TestUpdateLocation(t *testing.T) {
	testServer := &Server{saveLocation: ".testoutget", bridge: testBridge{}, org: &pb.Organisation{}}

	location := &pb.Location{
		Name:      "TestName",
		Units:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}
	log.Printf("BLAH %v", location.Sort)

	testServer.AddLocation(context.Background(), location)

	locationUpdate := &pb.Location{
		Name: "TestName",
		Sort: pb.Location_BY_DATE_ADDED,
	}
	testServer.UpdateLocation(context.Background(), locationUpdate)

	testServer2 := &Server{saveLocation: ".testoutget", bridge: testBridge{}, org: &pb.Organisation{}}
	testServer2.org = loadLatest(".testoutget")
	retrLocation := &pb.Location{Name: "TestName"}
	retr, err := testServer2.GetLocation(context.Background(), retrLocation)
	if err != nil {
		t.Errorf("Error on getting location: %v", err)
	}

	if retr.Name != "TestName" {
		t.Errorf("Location name is wrong: '%v' vs %v", retrLocation, location)
	}

	if len(retr.FolderIds) == 0 || retr.FolderIds[0] != 10 {
		t.Errorf("Folder Id has come back wrong: %v", retr)
	}

	//Check that we have re-orged as part of the UpdateLocation
	if retr.ReleasesLocation[0].ReleaseId != 2 {
		t.Errorf("Re-org has not occured: %v", retr)
	}
}

func TestGetLocationFail(t *testing.T) {
	testServer := &Server{saveLocation: ".testoutget", bridge: testBridge{}, org: &pb.Organisation{}}

	location2 := &pb.Location{
		Name:      "TestName",
		Units:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}

	testServer.AddLocation(context.Background(), location2)

	testServer2 := &Server{saveLocation: ".testoutget", bridge: testBridge{}, org: &pb.Organisation{}}
	testServer2.org = loadLatest(".testoutget")
	retrLocation := &pb.Location{Name: "MadeUpName"}
	_, err := testServer2.GetLocation(context.Background(), retrLocation)
	if err == nil {
		t.Errorf("Error on getting location: %v", err)
	}
}

func TestAddLocation(t *testing.T) {
	testServer := &Server{saveLocation: ".testaddalocation", bridge: testBridge{}, org: &pb.Organisation{}}

	location := &pb.Location{
		Name:      "TestName",
		Units:     2,
		FolderIds: []int32{10},
		Sort:      pb.Location_BY_LABEL_CATNO,
	}

	newLocation, err := testServer.AddLocation(context.Background(), location)
	if err != nil {
		t.Errorf("Error on adding location: %v", err)
	}

	if len(newLocation.ReleasesLocation) != 3 {
		t.Errorf("All the releases have not been organised")
	}

	if newLocation.ReleasesLocation[1].Slot != 2 {
		t.Errorf("Second release is in the wrong slot: %v", newLocation.ReleasesLocation[1])
	}

	if newLocation.ReleasesLocation[1].Index != 1 {
		t.Errorf("Second release has the wrong index %v", newLocation)
	}
}
