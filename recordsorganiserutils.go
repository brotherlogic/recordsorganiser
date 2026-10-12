package main

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbd "github.com/brotherlogic/godiscogs/proto"
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
	spill = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "recordsorganiser_spill",
	}, []string{"location"})

	salesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "recordsorganiser_sales_total",
		Help: "Count of records marked for sale",
	}, []string{"location"})

	saleErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "recordsorganiser_sale_errors_total",
		Help: "Count of errors encountered when preparing records for sale",
	}, []string{"location"})
)

var locationToGramophileOrg = map[string]string{
	"12 Inches": "12 Inches",
}

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
		go func(iid int64) {
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
		s.evaluateDarkSaleCandidate(ctx, c)
		records := []*pbrc.Record{}
		for _, rp := range c.GetReleasesLocation() {
			rec, err := s.bridge.getRecord(ctx, rp.GetInstanceId())
			if err != nil {
				return err
			}
			records = append(records, rec)
		}

		sort.Sort(sales.BySaleOrder(records))
	}

	return nil
}

func (s *Server) processSlotQuota(ctx context.Context, c *pb.Location) error {
	mslot := int32(0)
	cover := float64(0)
	for _, elem := range c.GetReleasesLocation() {
		if elem.GetSlot() > mslot {
			mslot = elem.GetSlot()
		}
		if elem.GetSlot() > c.GetQuota().GetSlots() {
			cover++
		}
	}

	foundSlots.With(prometheus.Labels{"org": c.GetName()}).Set(float64(mslot))
	spill.With(prometheus.Labels{"location": c.GetName()}).Set(cover)

	s.CtxLog(ctx, fmt.Sprintf("Found %v slots with a quota of %v for %v", mslot, c.GetQuota().GetSlots(), c.GetName()))

	if mslot > c.GetQuota().GetSlots() {
		s.evaluateDarkSaleCandidate(ctx, c)
		records := []*pbrc.Record{}
		for _, rp := range c.GetReleasesLocation() {
			rec, err := s.bridge.getRecord(ctx, rp.GetInstanceId())
			if err != nil {
				return err
			}
			records = append(records, rec)
		}

		sort.Sort(sales.BySaleOrder(records))
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
		_ = totalWidth
	}

	return nil
}

func (s *Server) processSaleCandidate(ctx context.Context, c *pb.Location) error {
	orgName, ok := locationToGramophileOrg[c.GetName()]
	if !ok {
		return nil
	}

	candidate, err := s.bridge.getSaleCandidate(ctx, orgName)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			s.CtxLog(ctx, fmt.Sprintf("No sale candidate found for %v (org: %v)", c.GetName(), orgName))
			return nil
		}
		s.CtxLog(ctx, fmt.Sprintf("Error retrieving sale candidate for %v (org: %v): %v", c.GetName(), orgName, err))
		saleErrorsTotal.With(prometheus.Labels{"location": c.GetName()}).Inc()
		return fmt.Errorf("failed to retrieve sale candidate for %v: %w", c.GetName(), err)
	}

	updateReq := &pbrc.UpdateRecordRequest{
		Reason: fmt.Sprintf("Organiser Sale: %v over quota", c.GetName()),
		Update: &pbrc.Record{
			Release: &pbd.Release{
				InstanceId: candidate.GetRecord().GetRelease().GetInstanceId(),
			},
			Metadata: &pbrc.ReleaseMetadata{
				Category: pbrc.ReleaseMetadata_PREPARE_TO_SELL,
			},
		},
	}

	_, err = s.bridge.updateRecord(ctx, updateReq)
	if err != nil {
		s.CtxLog(ctx, fmt.Sprintf("Error updating record %v to PREPARE_TO_SELL for %v: %v", candidate.GetRecord().GetRelease().GetInstanceId(), c.GetName(), err))
		saleErrorsTotal.With(prometheus.Labels{"location": c.GetName()}).Inc()
		return fmt.Errorf("failed to update record %v to PREPARE_TO_SELL: %w", candidate.GetRecord().GetRelease().GetInstanceId(), err)
	}

	s.CtxLog(ctx, fmt.Sprintf("Sale candidate prepared for %v: instance_id=%v, score=%v, median_price=%v",
		c.GetName(),
		candidate.GetRecord().GetRelease().GetInstanceId(),
		candidate.GetRecord().GetPackageScore(),
		candidate.GetRecord().GetMedianPrice(),
	))
	salesTotal.With(prometheus.Labels{"location": c.GetName()}).Inc()
	return nil
}

func (s *Server) evaluateDarkSaleCandidate(ctx context.Context, c *pb.Location) {
	_ = s.processSaleCandidate(ctx, c)
}

