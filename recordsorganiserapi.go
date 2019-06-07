package main

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/brotherlogic/goserver/utils"
	"github.com/golang/protobuf/proto"

	pb "github.com/brotherlogic/recordsorganiser/proto"
)

// UpdateLocation updates a given location
func (s *Server) UpdateLocation(ctx context.Context, req *pb.UpdateLocationRequest) (*pb.UpdateLocationResponse, error) {
	for i, loc := range s.org.GetLocations() {
		if loc.GetName() == req.GetLocation() {
			if req.DeleteLocation {
				s.org.Locations = append(s.org.GetLocations()[:i], s.org.GetLocations()[i+1:]...)
			} else {
				proto.Merge(loc, req.Update)
			}
		}
	}

	s.saveOrg(ctx)
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
	s.saveOrg(ctx)

	_, err := s.organise(s.org)

	if err != nil {
		return &pb.AddLocationResponse{}, err
	}

	return &pb.AddLocationResponse{Now: s.org}, nil
}

// GetOrganisation gets a given organisation
func (s *Server) GetOrganisation(ctx context.Context, req *pb.GetOrganisationRequest) (*pb.GetOrganisationResponse, error) {
	locations := make([]*pb.Location, 0)
	num := int32(0)

	for _, rloc := range req.GetLocations() {
		for _, loc := range s.org.GetLocations() {
			if utils.FuzzyMatch(rloc, loc) {
				if req.ForceReorg {
					n, err := s.organiseLocation(ctx, loc)
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
		s.saveOrg(ctx)
	}

	return &pb.GetOrganisationResponse{Locations: locations, NumberProcessed: num}, nil
}

// GetQuota fills out the quota response
func (s *Server) GetQuota(ctx context.Context, req *pb.QuotaRequest) (*pb.QuotaResponse, error) {
	st := time.Now()

	var loc *pb.Location
	for _, l := range s.org.GetLocations() {
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
		s.lastQuotaTime = time.Now().Sub(st)
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Unable to find folder with name '%v' or id %v", req.Name, req.FolderId))
	}

	recs := s.getRecordsForFolder(ctx, loc)
	s.Log(fmt.Sprintf("Getting Quota with %v records", len(recs)))
	instanceIDs := []int32{}
	for _, r := range recs {
		instanceIDs = append(instanceIDs, r.GetRelease().InstanceId)
	}

	//Old style quota
	if loc.GetQuota() != nil {
		if loc.GetQuota().NumOfSlots > 0 {
			if len(recs) > int(loc.GetQuota().NumOfSlots) {
				s.RaiseIssue(ctx, "Quota Problem", fmt.Sprintf("%v is over quota", loc.GetName()), false)
			}
			s.lastQuotaTime = time.Now().Sub(st)
			return &pb.QuotaResponse{OverQuota: len(recs) > int(loc.GetQuota().NumOfSlots), LocationName: loc.GetName(), InstanceId: instanceIDs, Quota: loc.GetQuota()}, nil
		}

		//New style quota
		if loc.GetQuota().GetSlots() > 0 {
			if len(recs) > int(loc.GetQuota().GetSlots()) {
				s.RaiseIssue(ctx, "Quota Problem", fmt.Sprintf("%v is over quota", loc.GetName()), false)
			}

			s.lastQuotaTime = time.Now().Sub(st)
			return &pb.QuotaResponse{OverQuota: len(recs) > int(loc.GetQuota().GetSlots()), LocationName: loc.GetName(), InstanceId: instanceIDs, Quota: loc.GetQuota()}, nil
		}

		// New Style quota part 2
		if loc.GetQuota().GetWidth() > 0 {
			totalWidth := int32(0)
			for _, r := range recs {
				if r.GetMetadata().SpineWidth <= 0 {
					s.RaiseIssue(ctx, "Missing Spine Width", fmt.Sprintf("Record %v is missing spine width (%v)", r.GetRelease().Title, r.GetRelease().Id), false)
					s.lastQuotaTime = time.Now().Sub(st)
					return nil, fmt.Errorf("Unable to compute quota - missing width")
				}
				totalWidth += r.GetMetadata().SpineWidth
			}
			if totalWidth > loc.GetQuota().GetWidth() {
				s.RaiseIssue(ctx, "Quota Problem", fmt.Sprintf("%v is over quota", loc.GetName()), false)
			}

			s.lastQuotaTime = time.Now().Sub(st)
			return &pb.QuotaResponse{OverQuota: totalWidth > loc.GetQuota().GetWidth(), LocationName: loc.GetName(), InstanceId: instanceIDs, Quota: loc.GetQuota()}, nil
		}
	}

	s.lastQuotaTime = time.Now().Sub(st)
	return &pb.QuotaResponse{}, status.Error(codes.InvalidArgument, fmt.Sprintf("No quota specified for location (%v)", loc.GetName()))
}

// AddExtractor adds an extractor
func (s *Server) AddExtractor(ctx context.Context, req *pb.AddExtractorRequest) (*pb.AddExtractorResponse, error) {
	s.org.Extractors = append(s.org.Extractors, req.Extractor)
	s.saveOrg(ctx)
	return &pb.AddExtractorResponse{}, nil
}
