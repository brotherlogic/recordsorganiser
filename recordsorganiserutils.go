package main

import (
	"fmt"

	"golang.org/x/net/context"

	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

func (s *Server) getRecordsForFolder(ctx context.Context, sloc *pb.Location) []*pbrc.Record {
	recs := []*pbrc.Record{}

	dirtyRecs, err := s.bridge.getReleasesWithGoal(ctx, sloc.FolderIds)
	if err != nil {
		return recs
	}

	// Get potential records from the listening pile
	for _, r := range dirtyRecs {
		for _, fid := range sloc.FolderIds {
			if r.GetMetadata().GoalFolder == fid {
				c := r.GetMetadata().Category
				if c != pbrc.ReleaseMetadata_UNLISTENED &&
					c != pbrc.ReleaseMetadata_STAGED &&
					c != pbrc.ReleaseMetadata_DIGITAL &&
					c != pbrc.ReleaseMetadata_STALE_SALES &&
					c != pbrc.ReleaseMetadata_LISTED_TO_SELL &&
					c != pbrc.ReleaseMetadata_STAGED_TO_SELL &&
					c != pbrc.ReleaseMetadata_SOLD &&
					c != pbrc.ReleaseMetadata_PREPARE_TO_SELL &&
					c != pbrc.ReleaseMetadata_HIGH_SCHOOL &&
					c != pbrc.ReleaseMetadata_NO_LABELS &&
					c != pbrc.ReleaseMetadata_SOLD_ARCHIVE &&
					c != pbrc.ReleaseMetadata_ASSESS_FOR_SALE &&
					c != pbrc.ReleaseMetadata_PRE_FRESHMAN {
					recs = append(recs, r)
				}
			}
		}
	}

	counts := make(map[string]int)
	cats := make(map[string]int)
	for _, r := range recs {
		cstr := fmt.Sprintf("%v", r.GetMetadata().GetCategory())
		if _, ok := counts[r.GetRelease().Title]; !ok {
			counts[r.GetRelease().Title] = 0
		}
		if _, ok := cats[cstr]; !ok {
			cats[cstr] = 0
		}
		cats[cstr]++
		counts[r.GetRelease().Title]++
	}

	for v, c := range cats {
		s.Log(fmt.Sprintf("%v %v -> %v", sloc.FolderIds, v, c))
	}

	done := false
	for v, c := range counts {
		if c > 1 {
			if !done {
				done = true
				s.Log(fmt.Sprintf("Double count on: %v", v))
			}
		}
	}

	return recs
}
