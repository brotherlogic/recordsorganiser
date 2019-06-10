package main

import (
	"fmt"

	pb "github.com/brotherlogic/recordsorganiser/proto"
	"golang.org/x/net/context"
)

func (s *Server) processQuota(ctx context.Context, c *pb.Location) error {
	slots := int(c.GetQuota().GetNumOfSlots())
	existing := len(c.ReleasesLocation)

	s.Log(fmt.Sprintf("Processing %v - selling %v records", c.Name, existing-slots))
	c.OverQuotaTime = 0
	return nil
}
