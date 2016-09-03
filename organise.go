package main

import (
	"errors"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"

	"github.com/brotherlogic/diffmove"
	"github.com/brotherlogic/goserver"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"

	pbs "github.com/brotherlogic/discogssyncer/server"
	pbd "github.com/brotherlogic/godiscogs"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

// Server the configuration for the syncer
type Server struct {
	*goserver.GoServer
	saveLocation string
	bridge       discogsBridge
	org          *pb.Organisation
}

type discogsBridge interface {
	getReleases(folders []int32) []*pbd.Release
	getRelease(ID int32) *pbd.Release
	getMetadata(release *pbd.Release) *pbs.ReleaseMetadata
}

func getMoves(start []*pb.ReleasePlacement, end []*pb.ReleasePlacement, slot int) []*pb.LocationMove {
	var moves []*pb.LocationMove

	inStartSlot := 0
	for _, startRec := range start {
		if int(startRec.Slot) == slot && int(startRec.Index) > inStartSlot {
			inStartSlot = int(startRec.Index)
		}
	}
	inEndSlot := 0
	for _, endRec := range end {
		if int(endRec.Slot) == slot && int(endRec.Index) > inEndSlot {
			inEndSlot = int(endRec.Index)
		}
	}

	//Build out the arrays for diffmove
	startNumbers := make([]int, inStartSlot+1)
	endNumbers := make([]int, inEndSlot+1)
	for _, startRec := range start {
		if int(startRec.Slot) == slot {
			startNumbers[startRec.Index-1] = int(startRec.ReleaseId)
		}
	}
	for _, endRec := range end {
		if int(endRec.Slot) == slot {
			endNumbers[endRec.Index-1] = int(endRec.ReleaseId)
		}
	}
	log.Printf("DIFFING %v -> %v", startNumbers, endNumbers)
	diffMoves := diffmove.Diff(startNumbers, endNumbers)
	log.Printf("RES: %v", diffMoves)
	for _, move := range diffMoves {
		switch move.Move {
		case "Add":
			moves = append(moves, &pb.LocationMove{
				New: &pb.ReleasePlacement{ReleaseId: int32(move.Value), Index: int32(move.Start + 1),
					BeforeReleaseId: int32(move.StartPrior), AfterReleaseId: int32(move.StartFollow)},
			})
		case "Delete":
			moves = append(moves, &pb.LocationMove{
				Old: &pb.ReleasePlacement{ReleaseId: int32(move.Value), Index: int32(move.Start + 1),
					BeforeReleaseId: int32(move.StartPrior), AfterReleaseId: int32(move.StartFollow)},
			})
			log.Printf("WHAT = %v from %v", moves, move)
		case "Move":
			moves = append(moves, &pb.LocationMove{
				Old: &pb.ReleasePlacement{ReleaseId: int32(move.Value), Index: int32(move.Start + 1),
					BeforeReleaseId: int32(move.StartPrior), AfterReleaseId: int32(move.StartFollow)},
				New: &pb.ReleasePlacement{ReleaseId: int32(move.Value), Index: int32(move.End + 1),
					BeforeReleaseId: int32(move.EndPrior), AfterReleaseId: int32(move.EndFollow)},
			})
		}
	}

	return moves
}

// Diff computes the diff between two slot organisations
func (s *Server) Diff(ctx context.Context, in *pb.DiffRequest) (*pb.OrganisationMoves, error) {

	orgStart, err := load(s.saveLocation, strconv.Itoa(int(in.StartTimestamp)))
	if err != nil {
		return nil, err
	}
	orgEnd, err := load(s.saveLocation, strconv.Itoa(int(in.EndTimestamp)))
	if err != nil {
		return nil, err
	}

	var locStart *pb.Location
	var locEnd *pb.Location
	for _, location := range orgStart.Locations {
		if location.Name == in.LocationName {
			locStart = location
		}
	}
	for _, location := range orgEnd.Locations {
		if location.Name == in.LocationName {
			locEnd = location
		}
	}

	if locEnd == nil || locStart == nil {
		return nil, errors.New("Unable to find location " + in.LocationName)
	}
	log.Printf("START = %v", locStart.ReleasesLocation)
	log.Printf("END = %v", locEnd.ReleasesLocation)
	moves := getMoves(locStart.ReleasesLocation, locEnd.ReleasesLocation, int(in.Slot))
	res := &pb.OrganisationMoves{
		StartTimestamp: in.StartTimestamp,
		EndTimestamp:   in.EndTimestamp,
		Moves:          moves,
	}
	return res, nil
}

// GetOrganisation Gets the current organisation
func (s *Server) GetOrganisation(ctx context.Context, in *pb.Empty) (*pb.Organisation, error) {
	return s.org, nil
}

// Locate gets the location of a given release
func (s *Server) Locate(ctx context.Context, in *pbd.Release) (*pb.ReleaseLocation, error) {
	relLoc := &pb.ReleaseLocation{}
	foundIndex := -1
	for _, loc := range s.org.Locations {
		for _, rel := range loc.ReleasesLocation {
			if rel.ReleaseId == in.Id {
				foundIndex = int(rel.Index)
				relLoc.Location = loc
				relLoc.Slot = rel.Slot
			}
		}
		if foundIndex >= 0 {
			for _, rel := range loc.ReleasesLocation {
				log.Printf("%v -> %v", rel, foundIndex)
				if rel.Slot == relLoc.Slot {
					if int(rel.Index) == foundIndex-1 {
						relLoc.Before = s.bridge.getRelease(rel.ReleaseId)
					}
					if int(rel.Index) == foundIndex+1 {
						relLoc.After = s.bridge.getRelease(rel.ReleaseId)
					}
				}
			}
			return relLoc, nil
		}
	}

	return nil, errors.New("Unable to locate id " + strconv.Itoa(int(in.Id)))
}

// GetOrganisations Gets all the available organisations
func (s *Server) GetOrganisations(ctx context.Context, in *pb.Empty) (*pb.OrganisationList, error) {
	orgList := &pb.OrganisationList{}
	files, _ := ioutil.ReadDir(s.saveLocation)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".data") {
			org, _ := load(s.saveLocation, file.Name()[0:len(file.Name())-5])
			orgList.Organisations = append(orgList.Organisations, org)
		}
	}

	return orgList, nil
}

func (s Server) runOrgSteps() {
	s.moveOldRecordsToPile()
}

func (s Server) moveOldRecordsToPile() {
	ip, port := getIP("discogssyncer", "10.0.1.17", 50055)
	conn, _ := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
	defer conn.Close()
	client := pbs.NewDiscogsServiceClient(conn)

	records, err := client.GetReleasesInFolder(context.Background(), &pbs.FolderList{Folders: []*pbd.Folder{&pbd.Folder{Id: 673768}}})
	if err != nil {
		panic(err)
	}

	for _, record := range records.Releases {
		meta, _ := client.GetMetadata(context.Background(), record)
		if meta.DateAdded < (time.Now().AddDate(0, -3, 0).Unix()) {
			if record.Rating > 0 {
				client.MoveToFolder(context.Background(), &pbs.ReleaseMove{Release: record, NewFolderId: 242017})
			} else {
				client.MoveToFolder(context.Background(), &pbs.ReleaseMove{Release: record, NewFolderId: 812802})
			}
		}
	}
}

// Organise Organises out the whole collection
func (s *Server) Organise(ctx context.Context, in *pb.Empty) (*pb.OrganisationMoves, error) {
	initialList := loadLatest(s.saveLocation)
	s.runOrgSteps()
	newList := &pb.Organisation{}

	for _, folder := range initialList.Locations {
		newList.Locations = append(newList.Locations, s.arrangeLocation(folder))
	}

	diffs := compare(initialList, newList)
	s.org = newList
	s.save()

	return &pb.OrganisationMoves{StartTimestamp: initialList.Timestamp, EndTimestamp: s.org.Timestamp, Moves: diffs}, nil
}

// GetLocation Gets an existing location
func (s *Server) GetLocation(ctx context.Context, location *pb.Location) (*pb.Location, error) {

	log.Printf("Now server: %v with %v", s, s.org)

	for _, storedLocation := range s.org.GetLocations() {
		if storedLocation.Name == location.Name {
			log.Printf("Returning %v", storedLocation)
			return storedLocation, nil
		}
	}

	return &pb.Location{}, errors.New("Cannot find location called " + location.Name)
}

func (s *Server) arrangeLocation(location *pb.Location) *pb.Location {
	releases := s.bridge.getReleases(location.FolderIds)
	retLocation := &pb.Location{Name: location.Name, Units: location.Units, FolderIds: location.FolderIds, Sort: location.Sort}

	switch location.Sort {
	case pb.Location_BY_LABEL_CATNO:
		sort.Sort(pbd.ByLabelCat(releases))
	case pb.Location_BY_DATE_ADDED:
		var combined []*pb.CombinedRelease
		for _, release := range releases {
			meta := s.bridge.getMetadata(release)
			comb := &pb.CombinedRelease{Release: release, Metadata: meta}
			combined = append(combined, comb)
		}
		sort.Sort(ByDateAdded(combined))
		newReleases := make([]*pbd.Release, len(releases))
		for i, comb := range combined {
			newReleases[i] = comb.Release
		}
		releases = newReleases
	case pb.Location_BY_RELEASE_DATE:
		sort.Sort(ByEarliestReleaseDate(releases))
	}
	splits := pbd.Split(releases, float64(location.Units))

	var locations []*pb.ReleasePlacement
	for i, split := range splits {
		for j, rel := range split {
			place := &pb.ReleasePlacement{
				ReleaseId: rel.Id,
				Index:     int32(j) + 1,
				Slot:      int32(i) + 1,
			}
			locations = append(locations, place)
		}
	}

	retLocation.ReleasesLocation = locations
	log.Printf("RETURNING: %v", retLocation)
	return retLocation
}

// AddLocation Adds a new location to the organiser
func (s *Server) AddLocation(ctx context.Context, location *pb.Location) (*pb.Location, error) {
	newLocation := s.arrangeLocation(location)
	log.Printf("Appending %v", newLocation)
	s.org.Locations = append(s.org.Locations, newLocation)
	log.Printf("Result %v", s.org)
	s.save()
	log.Printf("Saved %v from %v", s.org, s)
	return newLocation, nil
}

//UpdateLocation updates the location with new properties
func (s *Server) UpdateLocation(ctx context.Context, in *pb.Location) (*pb.Location, error) {
	log.Printf("Locations = %v but %v", s.org.Locations, in)
	for i, loc := range s.org.Locations {
		if loc.Name == in.Name {
			proto.Merge(loc, in)
			log.Printf("START ARRANGE %v -> %v", loc, in)
			newLocation := s.arrangeLocation(loc)
			log.Printf("NEW LOCATION: %v", newLocation)
			s.org.Locations[i] = newLocation
			s.save()
			return newLocation, nil
		}
	}
	return nil, errors.New("Cannot find location in org")
}
