package main

import (
	"fmt"
	"sort"
	"time"

	"golang.org/x/net/context"

	pbgd "github.com/brotherlogic/godiscogs/proto"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
	"github.com/brotherlogic/recordsorganiser/sales"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	getTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "recordsorganiser_get_time",
		Help: "Time take to organise a slot",
	}, []string{"folder"})
)

func (s *Server) getRecordsForFolder(ctx context.Context, sloc *pb.Location) []*pbrc.Record {
	t := time.Now()
	defer func() {
		getTime.With(prometheus.Labels{"folder": sloc.GetName()}).Observe(float64(time.Since(t).Milliseconds()))
	}()
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
		s.bridge.updateRecord(ctx, up)
	}
	return nil
}

func (s *Server) processAbsoluteWidthQuota(ctx context.Context, c *pb.Location) error {
	twidth := float32(0)

	for _, elem := range c.GetReleasesLocation() {
		twidth += elem.GetDeterminedWidth()
	}

	s.CtxLog(ctx, fmt.Sprintf("%v has Total width %v vs quota of %v", c.GetName(), twidth, c.GetQuota().GetAbsoluteWidth()))
	if twidth > c.GetQuota().GetAbsoluteWidth() {
		records := []*pbrc.Record{}
		for _, rp := range c.GetReleasesLocation() {
			rec, err := s.bridge.getRecord(ctx, rp.GetInstanceId())
			if err != nil {
				return err
			}
			records = append(records, rec)
		}

		sort.Sort(sales.BySaleOrder(records))

		// Find the first appropriate record
		r := records[0]
		for _, prec := range records {
			if prec.GetMetadata().GetBoxState() == pbrc.ReleaseMetadata_BOX_UNKNOWN ||
				prec.GetMetadata().GetBoxState() == pbrc.ReleaseMetadata_OUT_OF_BOX {
				r = prec
				break
			}
		}
		s.CtxLog(ctx, fmt.Sprintf("Attempting to sell: %v", r.GetRelease().GetInstanceId()))

		if r.GetMetadata().GetCategory() == pbrc.ReleaseMetadata_IN_COLLECTION {
			up := &pbrc.UpdateRecordRequest{Reason: "org-prepare-to-sell", Update: &pbrc.Record{Release: &pbgd.Release{InstanceId: r.GetRelease().InstanceId}, Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_PREPARE_TO_SELL}}}
			_, err := s.bridge.updateRecord(ctx, up)
			if err != nil {
				return err
			}
		}
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
			up := &pbrc.UpdateRecordRequest{Reason: "org-prepare-to-sell", Update: &pbrc.Record{Release: &pbgd.Release{InstanceId: records[pointer].GetRelease().InstanceId}, Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_PREPARE_TO_SELL}}}
			s.bridge.updateRecord(ctx, up)
			totalWidth -= records[pointer].GetMetadata().GetRecordWidth()
			pointer++
		}
	}

	return nil
}
