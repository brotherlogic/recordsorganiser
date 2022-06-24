package main

import (
	"log"
	"testing"

	"golang.org/x/net/context"

	pb "github.com/brotherlogic/recordsorganiser/proto"
)

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
	log.Printf("NEWLOC: %v", location)
	//testServer.org.Locations = append(testServer.org.Locations, location)

	f, err := testServer.Locate(context.Background(), &pb.LocateRequest{InstanceId: 12345})

	if err == nil {
		t.Fatalf("Failed locate has not failed: %v", f)
	}
}

type testgh struct{}

func (t *testgh) alert(ctx context.Context, r *pb.Location) error {
	return nil
}

func TestAddExtractor(t *testing.T) {
	testServer := getTestServer(".addExtractor")
	_, err := testServer.AddExtractor(context.Background(), &pb.AddExtractorRequest{Extractor: &pb.LabelExtractor{LabelId: 1234, Extractor: "hello"}})
	if err != nil {
		t.Errorf("Error adding label")
	}
}
