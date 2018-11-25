package main

import (
	"golang.org/x/net/context"

	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

func (s *Server) getRecordsForFolder(ctx context.Context, sloc *pb.Location) []*pbrc.Record {
	recs := []*pbrc.Record{}

	lPile, err := s.bridge.getReleases(ctx, []int32{812802})
	if err != nil {
		return recs
	}

	// Get potential records from the listening pile
	for _, r := range lPile {
		for _, fid := range sloc.FolderIds {
			if r.GetMetadata().GoalFolder == fid {
				c := r.GetMetadata().Category
				if c != pbrc.ReleaseMetadata_UNLISTENED &&
					c != pbrc.ReleaseMetadata_STAGED &&
					c != pbrc.ReleaseMetadata_STAGED_TO_SELL &&
					c != pbrc.ReleaseMetadata_SOLD &&
					c != pbrc.ReleaseMetadata_PREPARE_TO_SELL &&
					c != pbrc.ReleaseMetadata_PRE_FRESHMAN {
					recs = append(recs, r)
				}
			}
		}
	}

	for _, loc := range s.org.GetLocations() {
		if sloc.Name == loc.Name {
			for _, in := range loc.ReleasesLocation {
				r, err := s.bridge.getRecord(ctx, in.InstanceId)
				if err == nil {
					if r.GetMetadata().Category != pbrc.ReleaseMetadata_STAGED_TO_SELL &&
						r.GetMetadata().Category != pbrc.ReleaseMetadata_SOLD {
						recs = append(recs, r)
					}
				}
			}
		}
	}

	return recs
}
