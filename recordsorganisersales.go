package main

import (
	"fmt"
	"sort"
	"strings"

	pbgd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"

	"golang.org/x/net/context"
)

//BySaleOrder - the order in which we sell things
type BySaleOrder []*pbrc.Record

func getScore(r *pbrc.Record) int32 {
	if r.GetRelease().Rating != 0 {
		return r.GetRelease().Rating
	}
	return int32(r.GetMetadata().OverallScore)
}

func (a BySaleOrder) Len() int      { return len(a) }
func (a BySaleOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a BySaleOrder) Less(i, j int) bool {
	if a[i].GetMetadata() != nil && a[j].GetMetadata() != nil {
		if a[i].GetMetadata().Keep != a[j].GetMetadata().Keep {
			if a[i].GetMetadata().Keep == pbrc.ReleaseMetadata_KEEPER {
				return false
			}
			if a[j].GetMetadata().Keep == pbrc.ReleaseMetadata_KEEPER {
				return true
			}
		}
	}

	// Sort by score
	if a[i].GetMetadata() != nil && a[j].GetMetadata() != nil {
		if getScore(a[i]) != getScore(a[j]) {
			return getScore(a[i]) < getScore(a[j])
		}
	}

	if a[i].GetMetadata() != nil && a[j].GetMetadata() != nil {
		if a[i].GetMetadata().Match != a[j].GetMetadata().Match {
			if a[i].GetMetadata().Match == pbrc.ReleaseMetadata_FULL_MATCH {
				return true
			}
			if a[j].GetMetadata().Match == pbrc.ReleaseMetadata_FULL_MATCH {
				return false
			}
		}
	}

	if a[i].GetRelease().Released != a[j].GetRelease().Released {
		return a[i].GetRelease().Released > a[j].GetRelease().Released
	}

	return strings.Compare(a[i].GetRelease().Title, a[j].GetRelease().Title) < 0
}

func (s *Server) processQuota(ctx context.Context, c *pb.Location) error {
	slots := int(c.GetQuota().GetNumOfSlots())
	existing := len(c.ReleasesLocation)

	s.Log(fmt.Sprintf("Processing %v - selling %v records", c.Name, existing-slots))
	c.OverQuotaTime = 0

	records := []*pbrc.Record{}
	for _, rp := range c.ReleasesLocation {
		rec, err := s.bridge.getRecord(ctx, rp.InstanceId)
		if err != nil {
			return err
		}
		records = append(records, rec)
	}

	// Sort the record
	sort.Sort(BySaleOrder(records))

	for i := 0; i < existing-slots; i++ {
		up := &pbrc.UpdateRecordRequest{Update: &pbrc.Record{Release: &pbgd.Release{InstanceId: records[i].GetRelease().InstanceId}, Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_PREPARE_TO_SELL}}}
		s.bridge.updateRecord(ctx, up)
	}
	return nil
}
