package main

import (
	"fmt"
	"strings"
	"unicode"

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

func getEntry(c *pb.SortingCache, iid int32) *pb.CacheEntry {
	for _, elem := range c.GetCache() {
		if elem.GetInstanceId() == iid {
			return elem
		}
	}

	return nil
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

func convertCatno(catnoIn string) string {
	// Remove leading zeros
	catno := strings.TrimLeft(catnoIn, "0")

	ncat := ""
	in_bits := false
	previousWasLetter := true
	for _, r := range catno {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if previousWasLetter && unicode.IsNumber(r) {
				previousWasLetter = false
				ncat += " "
			} else if !previousWasLetter && unicode.IsLetter(r) {
				previousWasLetter = true
				ncat += " "
			}
			ncat += string(r)
			in_bits = false
		} else {
			if !in_bits {
				ncat += " "
				in_bits = true
			}
		}
	}

	for strings.Contains(ncat, "  ") {
		ncat = strings.ReplaceAll(ncat, "  ", " ")
	}

	return strings.TrimSpace(ncat)
}

func buildCacheEntry(rec *rcpb.Record) *pb.CacheEntry {
	label := gd.GetMainLabel(rec.GetRelease().GetLabels())
	labelString := ""
	seen := make(map[string]bool)
	for _, label := range rec.GetRelease().GetLabels() {
		if !seen[label.GetName()] {
			labelString += label.GetName()
			seen[label.GetName()] = true
		}
	}
	return &pb.CacheEntry{
		InstanceId: rec.GetRelease().GetInstanceId(),
		Width:      float64(rec.GetMetadata().GetRecordWidth()),
		Filled:     rec.GetMetadata().GetFiledUnder().String(),
		Folder:     rec.GetRelease().GetFolderId(),
		MainLabel:  label.GetName(),
		Category:   rec.GetMetadata().GetCategory().String(),
		Entry: map[string]string{
			"BY_LABEL":      strings.ToLower(label.GetName() + "|" + convertCatno(label.GetCatno()) + "|" + rec.GetRelease().GetTitle()),
			"BY_DATE_ADDED": strings.ToLower(fmt.Sprintf("%v", rec.GetMetadata().GetDateAdded()))},
	}
}
