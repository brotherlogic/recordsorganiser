package main

import (
	"google.golang.org/protobuf/proto"

	rcpb "github.com/brotherlogic/recordcollection/proto"
	ropb "github.com/brotherlogic/recordsorganiser/proto"
)

func labelMatch(r1, r2 *rcpb.Record) bool {
	for _, label1 := range r1.GetRelease().GetLabels() {
		for _, label2 := range r2.GetRelease().GetLabels() {
			if label1.GetName() == label2.GetName() {
				return true
			}
		}
	}
	return false
}

//For now this just collapses similar records down to a simple map
func collapse(records []*rcpb.Record, cache *ropb.SortingCache) ([]*rcpb.Record, map[int32][]*rcpb.Record) {
	mapper := make(map[int32][]*rcpb.Record)
	var nrecords []*rcpb.Record
	var trecord *rcpb.Record
	inlabel := false

	for i, rec := range records {
		if inlabel {
			if labelMatch(trecord, rec) {
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
				if labelMatch(rec, records[i+1]) {
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
