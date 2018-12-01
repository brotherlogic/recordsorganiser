package main

import (
	"fmt"

	"golang.org/x/net/context"

	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

func (s *Server) getRecordsForFolder(ctx context.Context, sloc *pb.Location) []*pbrc.Record {
	recs := []*pbrc.Record{}

	recs, err := s.bridge.getReleasesWithGoal(ctx, sloc.FolderIds)
	if err != nil {
		return recs
	}

	// Get potential records from the listening pile
	for _, r := range recs {
		for _, fid := range sloc.FolderIds {
			if r.GetMetadata().GoalFolder == fid {
				c := r.GetMetadata().Category
				if c != pbrc.ReleaseMetadata_UNLISTENED &&
					c != pbrc.ReleaseMetadata_STAGED &&
					c != pbrc.ReleaseMetadata_UNLISTENED &&
					c != pbrc.ReleaseMetadata_DIGITAL &&
					c != pbrc.ReleaseMetadata_STAGED_TO_SELL &&
					c != pbrc.ReleaseMetadata_SOLD &&
					c != pbrc.ReleaseMetadata_PREPARE_TO_SELL &&
					c != pbrc.ReleaseMetadata_PRE_FRESHMAN {
					recs = append(recs, r)
				}
			}
		}
	}

	s.Log(fmt.Sprintf("Found %v records from %v", len(recs), sloc.FolderIds))
	for i := 0; i < 10; i++ {
		s.Log(fmt.Sprintf("%v is %v", sloc.FolderIds, recs[i].GetRelease().Title))
	}

	return recs
}
