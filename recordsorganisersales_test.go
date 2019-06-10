package main

import (
	"testing"

	pb "github.com/brotherlogic/recordsorganiser/proto"

	"golang.org/x/net/context"
)

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
