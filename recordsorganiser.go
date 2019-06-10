package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/brotherlogic/goserver"
	"golang.org/x/net/context"

	pbs "github.com/brotherlogic/discogssyncer/server"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

// Server the configuration for the syncer
type Server struct {
	*goserver.GoServer
	bridge        discogsBridge
	org           *pb.Organisation
	gh            gh
	lastOrgTime   time.Duration
	lastOrgFolder string
	sortMap       map[int32]*pb.SortMapping
	lastQuotaTime time.Duration
	scNeeded      int64
}

type gh interface {
	alert(ctx context.Context, r *pb.Location) error
}

type discogsBridge interface {
	getReleases(ctx context.Context, folders []int32) ([]*pbrc.Record, error)
	getReleasesWithGoal(ctx context.Context, folders []int32) ([]*pbrc.Record, error)
	getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error)
	moveToFolder(releaseMove *pbs.ReleaseMove)
	GetIP(string) (string, int)
}

func (s *Server) prepareForReorg() {

}

func (s *Server) organise(c *pb.Organisation) (int32, error) {
	num := int32(0)
	for _, l := range s.org.Locations {
		n, err := s.organiseLocation(context.Background(), l)
		if err != nil {
			return -1, err
		}
		num += n
	}
	return num, nil
}

func convert(exs []*pb.LabelExtractor) map[int32]string {
	m := make(map[int32]string)
	for _, ex := range exs {
		m[ex.LabelId] = ex.Extractor
	}
	return m
}

func (s *Server) organiseLocation(ctx context.Context, c *pb.Location) (int32, error) {
	s.lastOrgFolder = c.Name
	fr, err := s.bridge.getReleases(ctx, c.GetFolderIds())

	if err != nil {
		return -1, err
	}

	switch c.GetSort() {
	case pb.Location_BY_DATE_ADDED:
		sort.Sort(ByDateAdded(fr))
	case pb.Location_BY_LABEL_CATNO:
		sort.Sort(ByLabelCat{fr, convert(s.org.GetExtractors()), s.Log})
	}

	records := s.Split(fr, float64(c.GetSlots()))
	c.ReleasesLocation = []*pb.ReleasePlacement{}
	if c.Checking == pb.Location_REQUIRE_STOCK_CHECK {
		s.scNeeded = 0
	}
	for slot, recs := range records {
		for i, rinloc := range recs {
			//Raise the alarm if a record needs a stock check
			if c.Checking == pb.Location_REQUIRE_STOCK_CHECK {
				if rinloc.GetMetadata().Keep != pbrc.ReleaseMetadata_KEEPER && rinloc.GetRelease().MasterId != 0 {
					if time.Now().Sub(time.Unix(rinloc.GetMetadata().LastStockCheck, 0)) > time.Hour*24*30*6 {
						s.scNeeded++
						s.RaiseIssue(ctx, "Stock Check Needed", fmt.Sprintf("%v is in need of a stock check", rinloc.GetRelease().Title), false)
					}
				}
			}

			c.ReleasesLocation = append(c.ReleasesLocation, &pb.ReleasePlacement{Slot: int32(slot + 1), Index: int32(i), InstanceId: rinloc.GetRelease().InstanceId, Title: rinloc.GetRelease().Title})
		}
	}

	if c.GetQuota().GetSlots() > 0 {
		if len(fr) > int(c.GetQuota().GetSlots()) {
			s.Log(fmt.Sprintf("%v is over quota", c.GetName()))
		} else {
			s.Log(fmt.Sprintf("%v us under quota (%v vs %v)", c.GetName(), len(fr), c.GetQuota().GetSlots()))
		}
	}

	if c.GetQuota().GetNumOfSlots() > 0 {
		if len(fr) > int(c.GetQuota().GetNumOfSlots()) {
			s.Log(fmt.Sprintf("%v is over slot quota", c.GetName()))
		} else {
			s.Log(fmt.Sprintf("%v us under slot quota (%v vs %v)", c.GetName(), len(fr), c.GetQuota().GetSlots()))
		}
	}

	return int32(len(fr)), nil
}

func (s *Server) checkQuota(ctx context.Context) error {
	for _, loc := range s.org.Locations {
		if loc.GetQuota() == nil && !loc.OptOutQuotaChecks {
			s.RaiseIssue(ctx, "Need Quota", fmt.Sprintf("%v needs to have some quota", loc.Name), false)
		}
	}
	return nil
}
