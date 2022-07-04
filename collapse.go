package main

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	rcpb "github.com/brotherlogic/recordcollection/proto"
	ropb "github.com/brotherlogic/recordsorganiser/proto"
)

func (s *Server) labelMatch(r1, r2 *rcpb.Record, cache *ropb.SortingCache) bool {
	for _, label1 := range r1.GetRelease().GetLabels() {
		for _, label2 := range r2.GetRelease().GetLabels() {
			if label1.GetName() == label2.GetName() {
				if getEntry(cache, r1.GetRelease().GetInstanceId()).GetLabelHash() != getEntry(cache, r2.GetRelease().GetInstanceId()).GetLabelHash() &&
					getEntry(cache, r1.GetRelease().GetInstanceId()).GetLabelHash() != "" {
					s.RaiseIssue("Bad label collab", fmt.Sprintf("%v VS %v", getEntry(cache, r1.GetRelease().GetInstanceId()), getEntry(cache, r2.GetRelease().GetInstanceId())))
				}
				return true
			}
		}
	}
	return false
}

//For now this just collapses similar records down to a simple map
func (s *Server) collapse(records []*rcpb.Record, cache *ropb.SortingCache) ([]*rcpb.Record, map[int32][]*rcpb.Record) {
	mapper := make(map[int32][]*rcpb.Record)
	var nrecords []*rcpb.Record
	var trecord *rcpb.Record
	inlabel := false

	for i, rec := range records {
		if inlabel {
			if s.labelMatch(trecord, rec, cache) {
				mapper[trecord.GetRelease().GetInstanceId()] = append(mapper[trecord.GetRelease().GetInstanceId()], rec)
				trecord.GetMetadata().RecordWidth += rec.GetMetadata().GetRecordWidth()
			} else {
				nrecords = append(nrecords, trecord)
				trecord = nil
				inlabel = false
			}
		}

		if !inlabel {
			if i < len(records)-1 {
				if s.labelMatch(rec, records[i+1], cache) {
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
