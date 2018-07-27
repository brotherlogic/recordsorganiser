package main

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/brotherlogic/goserver/utils"
	"github.com/golang/protobuf/proto"

	pbgd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
	pbt "github.com/brotherlogic/tracer/proto"
)

// UpdateLocation updates a given location
func (s *Server) UpdateLocation(ctx context.Context, req *pb.UpdateLocationRequest) (*pb.UpdateLocationResponse, error) {
	for _, loc := range s.org.GetLocations() {
		if loc.GetName() == req.GetLocation() {
			proto.Merge(loc, req.Update)
		}
	}

	s.saveOrg()
	return &pb.UpdateLocationResponse{}, nil
}

//Locate finds a record in the collection
func (s *Server) Locate(ctx context.Context, req *pb.LocateRequest) (*pb.LocateResponse, error) {
	for _, loc := range s.org.GetLocations() {
		for _, r := range loc.GetReleasesLocation() {
			if r.GetInstanceId() == req.GetInstanceId() {
				return &pb.LocateResponse{FoundLocation: loc}, nil
			}
		}
	}

	return &pb.LocateResponse{}, fmt.Errorf("Unable to locate %v in collection", req.GetInstanceId())
}

//AddLocation adds a location
func (s *Server) AddLocation(ctx context.Context, req *pb.AddLocationRequest) (*pb.AddLocationResponse, error) {
	s.prepareForReorg()
	s.org.Locations = append(s.org.Locations, req.GetAdd())
	s.saveOrg()

	_, err := s.organise(s.org)

	if err != nil {
		return &pb.AddLocationResponse{}, err
	}

	return &pb.AddLocationResponse{Now: s.org}, nil
}

// GetOrganisation gets a given organisation
func (s *Server) GetOrganisation(ctx context.Context, req *pb.GetOrganisationRequest) (*pb.GetOrganisationResponse, error) {
	ctx = s.LogTrace(ctx, "GetOrganisation", time.Now(), pbt.Milestone_START_FUNCTION)
	locations := make([]*pb.Location, 0)
	num := int32(0)

	for _, rloc := range req.GetLocations() {
		for _, loc := range s.org.GetLocations() {
			if utils.FuzzyMatch(rloc, loc) {
				if req.ForceReorg {
					n, err := s.organiseLocation(loc)
					num = n
					if err != nil {
						return &pb.GetOrganisationResponse{}, err
					}
				}
				locations = append(locations, loc)
			}
		}
	}

	if req.GetForceReorg() {
		s.saveOrg()
	}

	s.LogTrace(ctx, "GetOrganisation", time.Now(), pbt.Milestone_END_FUNCTION)
	return &pb.GetOrganisationResponse{Locations: locations, NumberProcessed: num}, nil
}

// GetQuota fills out the quota response
func (s *Server) GetQuota(ctx context.Context, req *pb.QuotaRequest) (*pb.QuotaResponse, error) {
	s.LogTrace(ctx, "GetQuota", time.Now(), pbt.Milestone_START_FUNCTION)
	t := time.Now()

	instanceIds := []int32{}

	folderIds := []int32{}
	for _, loc := range s.org.GetLocations() {
		if loc.Name == req.Name {
			folderIds = append(folderIds, loc.FolderIds...)
		}
	}

	if len(folderIds) == 0 {
		folderIds = append(folderIds, req.FolderId)
	}

	//Compute the count of valid records in the listening pile
	count := 0
	for _, loc := range s.org.GetLocations() {
		log.Printf("Trying %v", loc.Name)
		if loc.Name == "Listening Pile" {
			s.Log(fmt.Sprintf("Found %v", len(loc.ReleasesLocation)))
			for _, place := range loc.ReleasesLocation {
				meta, err := s.bridge.getMetadata(&pbgd.Release{InstanceId: place.InstanceId})
				log.Printf("META: %v", meta)
				if err == nil {
					for _, fid := range folderIds {
						if meta.GoalFolder == fid {
							if meta.Category != pbrc.ReleaseMetadata_UNLISTENED && meta.Category != pbrc.ReleaseMetadata_STAGED {
								if req.IncludeRecords {
									instanceIds = append(instanceIds, place.InstanceId)
								}
								count++
							} else {
								s.Log(fmt.Sprintf("Skipping %v", meta))
							}
						}
					}
				}
			}
		}
	}

	s.Log(fmt.Sprintf("COUNT = %v", count))
	for _, loc := range s.org.GetLocations() {
		for _, id := range loc.GetFolderIds() {
			if id == req.GetFolderId() || (req.Name == loc.Name) {
				s.Log(fmt.Sprintf("FOUND = %v,%v", len(loc.GetReleasesLocation()), loc.GetQuota().GetNumOfSlots()))
				if loc.GetQuota().GetNumOfSlots() > 0 && len(loc.GetReleasesLocation())+count >= int(loc.GetQuota().GetNumOfSlots()) {
					s.LogFunction("GetQuota-true", t)
					if !loc.GetNoAlert() {
						s.gh.alert(loc)
					}
					s.LogTrace(ctx, "GetQuota", time.Now(), pbt.Milestone_END_FUNCTION)
					if req.IncludeRecords {
						for _, in := range loc.ReleasesLocation {
							instanceIds = append(instanceIds, in.InstanceId)
						}
					}

					return &pb.QuotaResponse{SpillFolder: loc.SpillFolder, OverQuota: true, LocationName: loc.GetName(), InstanceId: instanceIds}, nil
				}

				s.LogFunction("GetQuota-false", t)
				s.LogTrace(ctx, "GetQuota", time.Now(), pbt.Milestone_END_FUNCTION)
				return &pb.QuotaResponse{OverQuota: false, LocationName: loc.GetName()}, nil
			}
		}
	}

	s.LogFunction("GetQuota-notfound", t)
	s.LogTrace(ctx, "GetQuota", time.Now(), pbt.Milestone_END_FUNCTION)
	return &pb.QuotaResponse{}, status.Error(codes.InvalidArgument, fmt.Sprintf("Unable to locate folder in request (%v)", req.GetFolderId()))
}

// AddExtractor adds an extractor
func (s *Server) AddExtractor(ctx context.Context, req *pb.AddExtractorRequest) (*pb.AddExtractorResponse, error) {
	s.org.Extractors = append(s.org.Extractors, req.Extractor)
	s.saveOrg()
	return &pb.AddExtractorResponse{}, nil
}
