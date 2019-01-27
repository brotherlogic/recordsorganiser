package main

import (
	"testing"

	"golang.org/x/net/context"

	pb "github.com/brotherlogic/recordsorganiser/proto"
)

func TestDeleteLocation(t *testing.T) {
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

	_, err = testServer.UpdateLocation(context.Background(), &pb.UpdateLocationRequest{Location: "TestName", DeleteLocation: true})
	if err != nil {
		t.Fatalf("Unable to update: %v", err)
	}

	resp, err = testServer.GetOrganisation(context.Background(), &pb.GetOrganisationRequest{ForceReorg: true, Locations: []*pb.Location{&pb.Location{Name: "TestName"}}})
	if err != nil {
		t.Fatalf("Unable to get organisation %v", err)
	}

	if len(resp.GetLocations()) != 0 {
		t.Errorf("Bad location response: %v", resp)
	}

}
