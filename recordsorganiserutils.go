package main

import (
	"fmt"
	"sort"

	"golang.org/x/net/context"

	pbgd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
	"github.com/brotherlogic/recordsorganiser/sales"
)

func (s *Server) getRecordsForFolder(ctx context.Context, sloc *pb.Location) []*pbrc.Record {
	recs := []*pbrc.Record{}

	ids, err := s.bridge.getReleases(ctx, sloc.FolderIds)
	if err != nil {
		return recs
	}

	// Get potential records from the listening pile
	for _, id := range ids {
		rec, err := s.bridge.getRecord(ctx, id)
		if err != nil {
			return recs
		}

		recs = append(recs, rec)
	}

	return recs
}

func (s *Server) processQuota(ctx context.Context, c *pb.Location) error {
	slots := int(c.GetQuota().GetNumOfSlots())
	existing := len(c.ReleasesLocation)

	s.Log(fmt.Sprintf("SALES:  since we're over numeric quota %v - selling %v records", c.Name, existing-slots))
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
	sort.Sort(sales.BySaleOrder(records))

	for i := 0; i < existing-slots; i++ {
		up := &pbrc.UpdateRecordRequest{Reason: "org-prepare-to-sell", Update: &pbrc.Record{Release: &pbgd.Release{InstanceId: records[i].GetRelease().InstanceId}, Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_PREPARE_TO_SELL}}}
		s.Log(fmt.Sprintf("Selling %v (%v)", records[i].GetRelease().Title, records[i].GetRelease().InstanceId))
		s.bridge.updateRecord(ctx, up)
	}
	return nil
}

func (s *Server) processWidthQuota(ctx context.Context, c *pb.Location) error {
	for slot := 0; slot <= int(c.GetSlots()); slot++ {
		totalWidth := float32(0)
		records := []*pbrc.Record{}
		for _, rp := range c.GetReleasesLocation() {
			if int(rp.GetSlot()) == slot {
				rec, err := s.bridge.getRecord(ctx, rp.GetInstanceId())
				if err != nil {
					return err
				}
				totalWidth += rec.GetMetadata().GetRecordWidth()
				records = append(records, rec)
			}
		}

		// Sort the record
		sort.Sort(sales.BySaleOrder(records))
		pointer := 0
		for pointer < len(records) && totalWidth > c.GetQuota().GetTotalWidth() {
			s.Log(fmt.Sprintf("SALES %v (slot %v of %v) because of width %v -> %v", records[pointer].GetRelease().GetInstanceId(), slot, c.GetName(), totalWidth, c.GetQuota().GetTotalWidth()))
			up := &pbrc.UpdateRecordRequest{Reason: "org-prepare-to-sell", Update: &pbrc.Record{Release: &pbgd.Release{InstanceId: records[pointer].GetRelease().InstanceId}, Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_PREPARE_TO_SELL}}}
			s.bridge.updateRecord(ctx, up)
			totalWidth -= records[pointer].GetMetadata().GetRecordWidth()
			pointer++
		}
	}

	return nil
}
