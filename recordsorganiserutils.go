package main

import (
	"fmt"
	"sort"
	"sync"
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

	foundSlots = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "recordsorganiser_found_slots",
	}, []string{"org"})
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
	wg := &sync.WaitGroup{}
	wg.Add(1)
	maxGoroutines := 100
	guard := make(chan struct{}, maxGoroutines)
	var ferr error
	for _, rp := range c.ReleasesLocation {
		guard <- struct{}{}
		wg.Add(1)
		go func(iid int32) {
			r, err := s.bridge.getRecord(ctx, iid)
			if err != nil {
				ferr = err
			} else {
				if !r.GetMetadata().GetNeedsGramUpdate() {
					found := false
					for _, folder := range c.GetFolderIds() {
						if folder == r.GetRelease().GetFolderId() {
							found = true
						}
					}
					if found {
						records = append(records, r)
					}
				}
			}
			wg.Done()
			<-guard
		}(rp.GetInstanceId())
	}
	wg.Done()
	wg.Wait()
	if ferr != nil {
		return ferr
	}

	// Sort the record
	sort.Sort(sales.BySaleOrder(records))

	if len(records) > 0 {
		for i := 0; i < existing-slots; i++ {
			s.CtxLog(ctx, fmt.Sprintf("Attempting with %v", records[i]))
			up := &pbrc.UpdateRecordRequest{Reason: "org-prepare-to-sell", Update: &pbrc.Record{Release: &pbgd.Release{InstanceId: records[i].GetRelease().InstanceId}, Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_PREPARE_TO_SELL}}}
			s.bridge.updateRecord(ctx, up)
		}
	}
	return nil
}

func (s *Server) processAbsoluteWidthQuota(ctx context.Context, c *pb.Location) error {
	twidth := float32(0)

	gwidth.With(prometheus.Labels{"location": c.GetName()}).Set(float64(c.GetQuota().GetAbsoluteWidth()))

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
		s.CtxLog(ctx, fmt.Sprintf("Attempting to force sell: %v", r.GetRelease().GetInstanceId()))

		up := &pbrc.UpdateRecordRequest{Reason: "org-prepare-to-sell", Update: &pbrc.Record{Release: &pbgd.Release{InstanceId: r.GetRelease().InstanceId}, Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_PREPARE_TO_SELL}}}
		_, err := s.bridge.updateRecord(ctx, up)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) processSlotQuota(ctx context.Context, c *pb.Location) error {
	mslot := int32(0)
	for _, elem := range c.GetReleasesLocation() {
		if elem.GetSlot() > mslot {
			mslot = elem.GetSlot()
		}
	}

	foundSlots.With(prometheus.Labels{"org": c.GetName()}).Set(float64(mslot))

	s.CtxLog(ctx, fmt.Sprintf("Found %v slots with a quota of %v for %v", mslot, c.GetQuota().GetSlots(), c.GetName()))

	if mslot > c.GetQuota().GetSlots() {
		records := []*pbrc.Record{}
		for _, rp := range c.GetReleasesLocation() {
			rec, err := s.bridge.getRecord(ctx, rp.GetInstanceId())
			if err != nil {
				return err
			}
			records = append(records, rec)
		}

		sort.Sort(sales.BySaleOrder(records))

		// Validate scores
		fscore := records[0].GetMetadata().GetOverallScore()
		diff := false
		for _, record := range records {
			if record.GetMetadata().GetOverallScore() != fscore {
				diff = true
			}
		}
		if !diff {
			s.RaiseIssue("Slot Stocked", fmt.Sprintf("%v is stocked", c.GetName()))
		}

		// Find the first appropriate record
		r := records[0]
		for _, prec := range records {
			if prec.GetMetadata().GetBoxState() == pbrc.ReleaseMetadata_BOX_UNKNOWN ||
				prec.GetMetadata().GetBoxState() == pbrc.ReleaseMetadata_OUT_OF_BOX {
				if !prec.GetMetadata().GetNeedsGramUpdate() {
					found := false
					for _, folder := range c.GetFolderIds() {
						if folder == prec.GetRelease().GetFolderId() {
							found = true
						}
					}
					if found {
						r = prec
					}
				}
				break
			}
		}
		s.CtxLog(ctx, fmt.Sprintf("Attempting to sell (%v): %v -> %v", c.GetName(), r.GetRelease().GetInstanceId(), r))

		up := &pbrc.UpdateRecordRequest{Reason: "org-prepare-to-sell", Update: &pbrc.Record{Release: &pbgd.Release{InstanceId: r.GetRelease().InstanceId}, Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_PREPARE_TO_SELL}}}
		_, err := s.bridge.updateRecord(ctx, up)
		if err != nil {
			return err
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
