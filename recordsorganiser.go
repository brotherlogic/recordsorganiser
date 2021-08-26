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
		Name: "recordsorganiser_size",
		Help: "Various Wait Times",
	}, []string{"location"})
)

func (s *Server) organiseLocation(ctx context.Context, c *pb.Location, org *pb.Organisation) (int32, error) {
	s.Log(fmt.Sprintf("Organising %v", c.GetName()))
	var overall []*pbrc.Record
	for i, _ := range c.GetFolderIds() {

		var lfold []int32
		var sorter pb.Location_Sorting
		for key, val := range c.GetFolderOrder() {
			if int(val) == i {
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
				if r.GetMetadata().Category == pbrc.ReleaseMetadata_ASSESS_FOR_SALE ||
					r.GetMetadata().Category == pbrc.ReleaseMetadata_PREPARE_TO_SELL ||
					r.GetMetadata().Category == pbrc.ReleaseMetadata_STAGED_TO_SELL {
					adjustment++
				}

				tfr = append(tfr, r)
			}
		}

		s.Log(fmt.Sprintf("SORT CODE: %v -> %v", lfold, sorter))
		switch sorter {
		case pb.Location_BY_DATE_ADDED:
			sort.Sort(ByDateAdded(tfr))
		case pb.Location_BY_LABEL_CATNO:
			sort.Sort(ByLabelCat{tfr, convert(org.GetExtractors()), s.Log})
		case pb.Location_BY_FOLDER_THEN_DATE:
			sort.Sort(ByFolderThenRelease(tfr))
		case pb.Location_BY_MOVE_TIME:
			sort.Sort(ByDateMoved(tfr))
		}

		overall = append(overall, tfr...)
		if overall != nil && len(overall) > 0 {
			s.Log(fmt.Sprintf("RUNNING APPEND: %v", overall[0]))
		}
	}

	records := s.Split(overall, float32(c.GetSlots()))
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

	sizes.With(prometheus.Labels{"location": c.GetName()}).Set(float64(len(overall)))
	return int32(len(overall)), s.saveOrg(ctx, org)
}
