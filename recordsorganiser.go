package main

import (
	"fmt"
	"sort"

	"github.com/brotherlogic/goserver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

// Server the configuration for the syncer
type Server struct {
	*goserver.GoServer
	bridge discogsBridge
}

type discogsBridge interface {
	getReleases(ctx context.Context, folders []int32) ([]int32, error)
	getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error)
	updateRecord(ctx context.Context, req *pbrc.UpdateRecordRequest) (*pbrc.UpdateRecordsResponse, error)
}

func convert(exs []*pb.LabelExtractor) map[int32]string {
	m := make(map[int32]string)
	for _, ex := range exs {
		m[ex.LabelId] = ex.Extractor
	}
	return m
}

func (s *Server) markOverQuota(ctx context.Context, c *pb.Location) error {
	if c.GetQuota().GetTotalWidth() > 0 {
		return s.processWidthQuota(ctx, c)
	}
	return s.processQuota(ctx, c)
}

var (
	sizes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "recordsorganiser_in_box",
		Help: "Various Wait Times",
	}, []string{"location"})

	swidths = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "recordsorgainser_slot_widths",
		Help: "Widthof slots",
	}, []string{"location", "slot"})
)

func (s *Server) organiseLocation(ctx context.Context, c *pb.Location, org *pb.Organisation) (int32, error) {
	s.Log(fmt.Sprintf("Organising the %v", c.GetName()))
	var overall []*pbrc.Record
	boxCount := 0
	var gaps []int
	widths := make(map[int32]float64)
	for ind, i := range c.GetFolderIds() {
		if ind > 0 && c.GetHardGap()[i] {
			gaps = append(gaps, len(overall))
		}

		var lfold []int32
		var sorter pb.Location_Sorting
		for key, val := range c.GetFolderOrder() {
			if val == int32(ind) {
				lfold = append(lfold, key)
				sorter = c.GetFolderSort()[key]
			}
		}

		ids, err := s.bridge.getReleases(ctx, lfold)
		if err != nil {
			return -1, err
		}

		adjustment := 0
		tfr := []*pbrc.Record{}
		for _, id := range ids {
			r, err := s.bridge.getRecord(ctx, id)
			if status.Convert(err).Code() != codes.OutOfRange {
				if err != nil {
					return -1, err
				}
				widths[r.GetRelease().GetInstanceId()] = float64(r.GetMetadata().GetRecordWidth())
				if r.GetMetadata().Category == pbrc.ReleaseMetadata_ASSESS_FOR_SALE ||
					r.GetMetadata().Category == pbrc.ReleaseMetadata_PREPARE_TO_SELL ||
					r.GetMetadata().Category == pbrc.ReleaseMetadata_STAGED_TO_SELL {
					adjustment++
				}

				tfr = append(tfr, r)
			}

			if r.GetMetadata().GetBoxState() != pbrc.ReleaseMetadata_BOX_UNKNOWN && r.GetMetadata().GetBoxState() != pbrc.ReleaseMetadata_OUT_OF_BOX {
				boxCount++
			}
		}

		switch sorter {
		case pb.Location_BY_DATE_ADDED:
			sort.Sort(ByDateAdded(tfr))
		case pb.Location_BY_LABEL_CATNO:
			sort.Sort(ByLabelCat{tfr, convert(org.GetExtractors()), s.Log})
		case pb.Location_BY_FOLDER_THEN_DATE:
			sort.Sort(ByFolderThenRelease(tfr))
		case pb.Location_BY_MOVE_TIME:
			sort.Sort(ByDateMoved(tfr))
		case pb.Location_BY_LAST_LISTEN:
			sort.Sort(ByLastListen(tfr))
		}

		overall = append(overall, tfr...)
	}

	s.Log(fmt.Sprintf("Running split with %v and %v", gaps, len(overall)))
	records := s.Split(overall, float32(c.GetSlots()), float32(c.GetQuota().GetTotalWidth()), gaps)
	c.ReleasesLocation = []*pb.ReleasePlacement{}
	for slot, recs := range records {
		for i, rinloc := range recs {
			c.ReleasesLocation = append(c.ReleasesLocation, &pb.ReleasePlacement{Slot: int32(slot + 1), Index: int32(i), InstanceId: rinloc.GetRelease().InstanceId, Title: rinloc.GetRelease().Title})
		}
	}

	//Make any quota adjustments - we only do width ajdustments
	if c.GetQuota().GetTotalWidth() > 0 {
		s.markOverQuota(ctx, c)
	}

	slotWidths := make(map[int]float64)
	for _, ent := range c.GetReleasesLocation() {
		slotWidths[int(ent.GetSlot())] += widths[ent.GetInstanceId()]
	}

	for slot, width := range slotWidths {
		swidths.With(prometheus.Labels{"location": c.GetName(), "slot": fmt.Sprintf("%v", slot)}).Set(width)
	}

	sizes.With(prometheus.Labels{"location": c.GetName()}).Set(float64((boxCount)))
	return int32(len(overall)), s.saveOrg(ctx, org)
}
