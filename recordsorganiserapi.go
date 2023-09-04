package main

import (
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/proto"

	pbgd "github.com/brotherlogic/godiscogs/proto"
	rcpb "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

// UpdateLocation updates a given location
func (s *Server) UpdateLocation(ctx context.Context, req *pb.UpdateLocationRequest) (*pb.UpdateLocationResponse, error) {
	org, err := s.readOrg(ctx)
	if err != nil {
		return nil, err
	}

	for i, loc := range org.GetLocations() {
		if loc.GetName() == req.GetLocation() {
			if req.DeleteLocation {
				org.Locations = append(org.GetLocations()[:i], org.GetLocations()[i+1:]...)
			} else {
				proto.Merge(loc, req.Update)
			}
		}
	}

	return &pb.UpdateLocationResponse{}, s.saveOrg(ctx, org)
}

// Locate finds a record in the collection
func (s *Server) Locate(ctx context.Context, req *pb.LocateRequest) (*pb.LocateResponse, error) {
	org, err := s.readOrg(ctx)
	if err != nil {
		return nil, err
	}

	for _, loc := range org.GetLocations() {
		for _, r := range loc.GetReleasesLocation() {
			if r.GetInstanceId() == req.GetInstanceId() {
				return &pb.LocateResponse{FoundLocation: loc}, nil
			}
		}
		for _, f := range loc.GetFolderIds() {
			if f == req.GetFolderId() {
				return &pb.LocateResponse{FoundLocation: loc}, nil
			}
		}
	}

	return &pb.LocateResponse{}, status.Errorf(codes.NotFound, "Unable to locate %v in collection", req.GetInstanceId())
}

// AddLocation adds a location
func (s *Server) AddLocation(ctx context.Context, req *pb.AddLocationRequest) (*pb.AddLocationResponse, error) {
	org, err := s.readOrg(ctx)
	if err != nil {
		return nil, err
	}

	cache, err := s.loadCache(ctx)
	if err != nil {
		return nil, err
	}

	org.Locations = append(org.Locations, req.GetAdd())
	err = s.saveOrg(ctx, org)
	if err != nil {
		return nil, err
	}

	_, err = s.organiseLocation(ctx, cache, req.GetAdd(), org)
	if err != nil {
		return nil, err
	}

	return &pb.AddLocationResponse{Now: org}, nil
}

func (s *Server) GetCache(ctx context.Context, _ *pb.GetCacheRequest) (*pb.GetCacheResponse, error) {
	cache, err := s.loadCache(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.GetCacheResponse{Cache: cache}, nil
}

func (s *Server) metrics(ctx context.Context) error {
	org, err := s.readOrg(ctx)
	if err != nil {
		return err
	}

	cache, err := s.loadCache(ctx)
	if err != nil {
		return err
	}

	for _, loc := range org.GetLocations() {
		s.organiseLocation(ctx, cache, loc, org)
	}

	return nil
}

// GetOrganisation gets a given organisation
func (s *Server) GetOrganisation(ctx context.Context, req *pb.GetOrganisationRequest) (*pb.GetOrganisationResponse, error) {
	org, err := s.readOrg(ctx)
	if err != nil {
		return nil, err
	}

	cache, err := s.loadCache(ctx)
	if err != nil {
		return nil, err
	}

	locations := make([]*pb.Location, 0)
	num := int32(0)

	if len(req.GetLocations()) == 0 {
		locations = org.GetLocations()
	}

	for _, rloc := range req.GetLocations() {
		for _, loc := range org.GetLocations() {
			if rloc.GetName() == loc.GetName() || rloc.GetName() == "" {
				if req.ForceReorg {
					n, err := s.organiseLocation(ctx, cache, loc, org)
					num = n
					if err != nil {
						return &pb.GetOrganisationResponse{}, err
					}
				}

				if req.OrgReset {
					loc.LastReorg = time.Now().Unix()
				}

				locations = append(locations, loc)
			}
		}
	}

	if req.GetForceReorg() {
		err := s.saveOrg(ctx, org)
		if err != nil {
			return nil, err
		}

		err = s.saveCache(ctx, cache)
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetOrganisationResponse{Locations: locations, NumberProcessed: num}, nil
}

// GetQuota fills out the quota response
func (s *Server) GetQuota(ctx context.Context, req *pb.QuotaRequest) (*pb.QuotaResponse, error) {
	org, err := s.readOrg(ctx)
	if err != nil {
		return nil, err
	}

	var loc *pb.Location
	for _, l := range org.GetLocations() {
		if l.Name == req.Name {
			loc = l
		}
		for _, id := range l.FolderIds {
			if id == req.FolderId {
				loc = l
			}
		}
	}

	if loc == nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Unable to find folder with name '%v' or id %v", req.Name, req.FolderId))
	}

	recs := s.getRecordsForFolder(ctx, loc)
	instanceIDs := []int32{}
	for _, r := range recs {
		instanceIDs = append(instanceIDs, r.GetRelease().InstanceId)
	}

	//Old style quota
	if loc.GetQuota() != nil {
		if loc.GetQuota().NumOfSlots > 0 {
			if len(recs) > int(loc.GetQuota().NumOfSlots) {
				s.RaiseIssue("Quota Problem", fmt.Sprintf("%v is over quota", loc.GetName()))
			}
			return &pb.QuotaResponse{OverQuota: len(recs) > int(loc.GetQuota().NumOfSlots), LocationName: loc.GetName(), InstanceId: instanceIDs, Quota: loc.GetQuota()}, nil
		}

		//New style quota
		if loc.GetQuota().GetSlots() > 0 {
			if len(recs) > int(loc.GetQuota().GetSlots()) {
				s.RaiseIssue("Quota Problem", fmt.Sprintf("%v is over quota", loc.GetName()))
			}

			return &pb.QuotaResponse{OverQuota: len(recs) > int(loc.GetQuota().GetSlots()), LocationName: loc.GetName(), InstanceId: instanceIDs, Quota: loc.GetQuota()}, nil
		}

		// New Style quota part 2
		if loc.GetQuota().GetWidth() > 0 {
			totalWidth := float32(0)
			for _, r := range recs {
				if r.GetMetadata().RecordWidth <= 0 {
					s.RaiseIssue("Missing Spine Width", fmt.Sprintf("Record %v is missing spine width (%v)", r.GetRelease().Title, r.GetRelease().Id))
					return nil, fmt.Errorf("Unable to compute quota - missing width")
				}
				totalWidth += r.GetMetadata().RecordWidth
			}
			if totalWidth > loc.GetQuota().GetWidth() {
				s.RaiseIssue("Quota Problem", fmt.Sprintf("%v is over quota", loc.GetName()))
			}

			return &pb.QuotaResponse{OverQuota: totalWidth > loc.GetQuota().GetWidth(), LocationName: loc.GetName(), InstanceId: instanceIDs, Quota: loc.GetQuota()}, nil
		}
	}

	return &pb.QuotaResponse{}, status.Error(codes.InvalidArgument, fmt.Sprintf("No quota specified for location (%v)", loc.GetName()))
}

// AddExtractor adds an extractor
func (s *Server) AddExtractor(ctx context.Context, req *pb.AddExtractorRequest) (*pb.AddExtractorResponse, error) {
	org, err := s.readOrg(ctx)
	if err != nil {
		return nil, err
	}

	org.Extractors = append(org.Extractors, req.GetExtractor())
	return &pb.AddExtractorResponse{}, s.saveOrg(ctx, org)
}

// ClientUpdate on an updated record
func (s *Server) ClientUpdate(ctx context.Context, req *rcpb.ClientUpdateRequest) (*rcpb.ClientUpdateResponse, error) {
	org, err := s.readOrg(ctx)
	if err != nil {
		return nil, err
	}

	// Post the reorgs
	for _, loc := range org.GetLocations() {
		if loc.GetName() == "12 Inches" {
			cyear := time.Now().YearDay()
			if cyear != int(loc.GetLastSort()) {
				if len(loc.GetSlotsToSort()) == 0 {
					slots := make(map[int32]bool)
					for _, entry := range loc.GetReleasesLocation() {
						slots[entry.GetSlot()] = true
					}

					for slotv := range slots {
						loc.SlotsToSort = append(loc.SlotsToSort, slotv)
					}

					rand.Seed(time.Now().UnixNano())
					rand.Shuffle(len(loc.SlotsToSort), func(i, j int) {
						loc.SlotsToSort[i], loc.SlotsToSort[j] = loc.SlotsToSort[j], loc.SlotsToSort[i]
					})
				}

				loc.LastSort = int32(cyear)
				//slot := loc.SlotsToSort[0]
				//s.RaiseIssue("Reorg 12 Inches", fmt.Sprintf("Slot %v needs a sort", slot))
				loc.SlotsToSort = loc.SlotsToSort[1:]
				s.saveOrg(ctx, org)
			}
		}
	}

	record, err := s.bridge.getRecord(ctx, req.GetInstanceId())
	if err != nil {
		if status.Convert(err).Code() == codes.OutOfRange {
			return &rcpb.ClientUpdateResponse{}, nil
		}
		return nil, err
	}

	oldLoc := &pb.Location{}
	newLoc := &pb.Location{}
	for _, loc := range org.GetLocations() {
		for _, place := range loc.GetReleasesLocation() {
			if place.GetInstanceId() == record.GetRelease().GetInstanceId() {
				oldLoc = loc
			}
		}

		for _, folder := range loc.GetFolderIds() {
			if folder == record.GetRelease().GetFolderId() {
				newLoc = loc
			}
		}
	}

	if oldLoc.GetName() != newLoc.GetName() {
		// Update the cache
		cache, err := s.updateCache(ctx, record)
		if err != nil {
			return nil, err
		}

		if len(oldLoc.GetName()) > 0 {
			_, err := s.organiseLocation(ctx, cache, oldLoc, org)
			if err != nil {
				return nil, err
			}
		} else {
			time.Sleep(time.Second * 2)
		}

		if len(newLoc.GetName()) > 0 {
			_, err := s.organiseLocation(ctx, cache, newLoc, org)
			if err != nil {
				return nil, err
			}
		}

		if len(oldLoc.GetName()) > 0 || len(newLoc.GetName()) > 0 {
			if record.GetMetadata().GetBoxState() != rcpb.ReleaseMetadata_IN_THE_BOX {
				_, err := s.bridge.updateRecord(ctx, &rcpb.UpdateRecordRequest{Reason: fmt.Sprintf("Org Move Update (%v -> %v)", oldLoc.GetName(), newLoc.GetName()), Update: &rcpb.Record{Release: &pbgd.Release{InstanceId: record.GetRelease().GetInstanceId()}}})
				return &rcpb.ClientUpdateResponse{}, err
			}
		}

		err = s.saveCache(ctx, cache)
		if err != nil {
			return nil, err
		}
	}

	return &rcpb.ClientUpdateResponse{}, nil
}
