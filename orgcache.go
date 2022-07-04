package main

import (
	"fmt"
	"strings"

	"golang.org/x/net/context"

	gd "github.com/brotherlogic/godiscogs"
	rcpb "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

func (s *Server) updateCache(ctx context.Context, rec *rcpb.Record) (*pb.SortingCache, error) {

	cache, err := s.loadCache(ctx)
	if err != nil {
		return nil, err
	}

	appendCache(cache, rec)

	return cache, s.saveCache(ctx, cache)
}

func appendCache(cache *pb.SortingCache, rec *rcpb.Record) *pb.CacheEntry {
	cacheEntry := buildCacheEntry(rec)

	var entries []*pb.CacheEntry
	for _, entry := range cache.GetCache() {
		if entry.GetInstanceId() != cacheEntry.GetInstanceId() {
			entries = append(entries, entry)
		}
	}
	entries = append(entries, cacheEntry)
	cache.Cache = entries
	return cacheEntry
}

func buildCacheEntry(rec *rcpb.Record) *pb.CacheEntry {
	label := gd.GetMainLabel(rec.GetRelease().GetLabels())
	labelString := ""
	for _, label := range rec.GetRelease().GetLabels() {
		labelString += label.GetName()
	}
	return &pb.CacheEntry{
		InstanceId: rec.GetRelease().GetInstanceId(),
		Width:      float64(rec.GetMetadata().GetRecordWidth()),
		Filled:     rec.GetMetadata().GetFiledUnder().String(),
		Folder:     rec.GetRelease().GetFolderId(),
		LabelHash:  labelString,
		Entry: map[string]string{
			"BY_LABEL":      strings.ToLower(label.GetName() + "-" + label.GetCatno()),
			"BY_DATE_ADDED": strings.ToLower(fmt.Sprintf("%v", rec.GetMetadata().GetDateAdded()))},
	}
}