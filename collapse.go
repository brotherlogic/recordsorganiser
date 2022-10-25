package main

import (
	"fmt"

	"golang.org/x/net/context"
	"google.golang.org/protobuf/proto"

	rcpb "github.com/brotherlogic/recordcollection/proto"
	ropb "github.com/brotherlogic/recordsorganiser/proto"
)

func (s *Server) labelMatch(ctx context.Context, r1, r2 *rcpb.Record, cache *ropb.SortingCache) bool {
	for _, label1 := range r1.GetRelease().GetLabels() {
		for _, label2 := range r2.GetRelease().GetLabels() {
			if label1.GetName() == label2.GetName() {
				if getEntry(cache, r1.GetRelease().GetInstanceId()).GetMainLabel() != getEntry(cache, r2.GetRelease().GetInstanceId()).GetMainLabel() &&
					getEntry(cache, r1.GetRelease().GetInstanceId()).GetMainLabel() != "" {
					s.CtxLog(ctx, fmt.Sprintf("Raising because %v and %v", r1, r2))
					s.RaiseIssue("Bad label collab", fmt.Sprintf("%v VS %v", getEntry(cache, r1.GetRelease().GetInstanceId()), getEntry(cache, r2.GetRelease().GetInstanceId())))
				}
				return true
			}
		}
	}
	return false
}

func (s *Server) adjust(width float32, mSleeve, dSleeve rcpb.ReleaseMetadata_SleeveState) float32 {
	if mSleeve == dSleeve {
		return width
	}

	if mSleeve == rcpb.ReleaseMetadata_BOX_SET && dSleeve == rcpb.ReleaseMetadata_VINYL_STORAGE_DOUBLE_FLAP {
		return width * 1.26
	}
	if mSleeve == rcpb.ReleaseMetadata_VINYL_STORAGE_DOUBLE_FLAP && dSleeve == rcpb.ReleaseMetadata_VINYL_STORAGE_NO_INNER {
		return width * (1.4 / 1.26)
	}
	if mSleeve == rcpb.ReleaseMetadata_VINYL_STORAGE_DOUBLE_FLAP && dSleeve == rcpb.ReleaseMetadata_BOX_SET {
		return width * (1 / 1.26)
	}
	if mSleeve == rcpb.ReleaseMetadata_VINYL_STORAGE_NO_INNER && dSleeve == rcpb.ReleaseMetadata_VINYL_STORAGE_DOUBLE_FLAP {
		return width * (1.26 / 1.4)
	}
	if mSleeve == rcpb.ReleaseMetadata_SLEEVE_UNKNOWN && dSleeve == rcpb.ReleaseMetadata_BOX_SET {
		return width * (1 / 1.18)
	}

	s.RaiseIssue("Sleeve mismatch", fmt.Sprintf("%v -> %v", mSleeve, dSleeve))

	return width
}

// For now this just collapses similar records down to a simple map
func (s *Server) collapse(ctx context.Context, records []*rcpb.Record, cache *ropb.SortingCache) ([]*rcpb.Record, map[int32][]*rcpb.Record) {
	mapper := make(map[int32][]*rcpb.Record)
	var nrecords []*rcpb.Record
	var trecord *rcpb.Record
	inlabel := false

	for i, rec := range records {
		if inlabel {
			if s.labelMatch(ctx, trecord, rec, cache) {
				mapper[trecord.GetRelease().GetInstanceId()] = append(mapper[trecord.GetRelease().GetInstanceId()], rec)
				trecord.GetMetadata().RecordWidth += s.adjust(rec.GetMetadata().GetRecordWidth(), trecord.GetMetadata().GetSleeve(), rec.GetMetadata().GetSleeve())
			} else {
				nrecords = append(nrecords, trecord)
				trecord = nil
				inlabel = false
			}
		}

		if !inlabel {
			if i < len(records)-1 {
				if s.labelMatch(ctx, rec, records[i+1], cache) {
					inlabel = true
					temp := proto.Clone(rec)
					trecord = temp.(*rcpb.Record)
					mapper[rec.GetRelease().GetInstanceId()] = []*rcpb.Record{rec}
				} else {
					nrecords = append(nrecords, rec)
				}
			} else {
				nrecords = append(nrecords, rec)
			}
		}
	}
	return nrecords, mapper
}

func expand(records []*rcpb.Record, mapper map[int32][]*rcpb.Record) []*rcpb.Record {
	var nrecords []*rcpb.Record

	for _, r := range records {
		if val, ok := mapper[r.GetRelease().GetInstanceId()]; ok {
			nrecords = append(nrecords, val...)
		} else {
			nrecords = append(nrecords, r)
		}
	}

	return nrecords
}
