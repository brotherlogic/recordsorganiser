package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"github.com/brotherlogic/goserver"
	"github.com/brotherlogic/keystore/client"
	pbt "github.com/brotherlogic/tracer/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pbs "github.com/brotherlogic/discogssyncer/server"
	pbgh "github.com/brotherlogic/githubcard/proto"
	pbd "github.com/brotherlogic/godiscogs"
	pbgs "github.com/brotherlogic/goserver/proto"
	"github.com/brotherlogic/goserver/utils"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

type prodGh struct{}

func (gh *prodGh) alert(ctx context.Context, r *pb.Location) error {
	host, port, err := utils.Resolve("githubcard")

	if err != nil {
		return err
	}

	conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
	defer conn.Close()
	if err != nil {
		return err
	}

	client := pbgh.NewGithubClient(conn)
	_, err = client.AddIssue(ctx, &pbgh.Issue{Title: "Quota Issue", Body: fmt.Sprintf("%v is out of quota", r.GetName()), Service: "recordsorganiser"})
	return err
}

// Bridge that accesses discogs syncer server
type prodBridge struct {
	Resolver func(string) (string, int)
}

const (
	//KEY is where we store the org
	KEY = "github.com/brotherlogic/recordsorganiser/org"
)

func (s *Server) readOrg(ctx context.Context) error {
	org := &pb.Organisation{}
	data, _, err := s.KSclient.Read(ctx, KEY, org)

	if err != nil {
		return err
	}
	s.org = data.(*pb.Organisation)

	// Build out the mapping
	for _, mapping := range s.org.SortMappings {
		s.sortMap[mapping.InstanceId] = mapping
	}

	return nil
}

func (s *Server) saveOrg(ctx context.Context) {
	//Unravel the mapping
	s.org.SortMappings = []*pb.SortMapping{}
	for _, mapping := range s.sortMap {
		s.org.SortMappings = append(s.org.SortMappings, mapping)
	}

	s.KSclient.Save(ctx, KEY, s.org)
}

func (discogsBridge prodBridge) GetIP(name string) (string, int) {
	return discogsBridge.Resolver(name)
}

func (discogsBridge prodBridge) getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error) {
	ip, port := discogsBridge.GetIP("recordcollection")
	conn, err2 := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
	if err2 == nil {
		defer conn.Close()
		client := pbrc.NewRecordCollectionServiceClient(conn)
		meta, err3 := client.GetRecords(ctx, &pbrc.GetRecordsRequest{Filter: &pbrc.Record{Release: &pbd.Release{InstanceId: instanceID}}})
		if err3 == nil && meta != nil && len(meta.Records) == 1 && meta.Records[0].Metadata != nil {
			return meta.Records[0], nil
		}
		return nil, fmt.Errorf("Problem getting meta %v and %v", err3, meta)
	}

	return nil, fmt.Errorf("Unable to get release metadata: %v", err2)
}

func (discogsBridge prodBridge) moveToFolder(move *pbs.ReleaseMove) {
	ip, port := discogsBridge.GetIP("discogssyncer")
	conn, _ := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
	defer conn.Close()
	client := pbs.NewDiscogsServiceClient(conn)
	client.MoveToFolder(context.Background(), move)
}

func (discogsBridge prodBridge) getReleases(ctx context.Context, folders []int32) ([]*pbrc.Record, error) {
	ctx = utils.Trace(ctx, "getReleases", time.Now(), pbt.Milestone_START_FUNCTION, "recordsorganiser")
	var result []*pbrc.Record

	for _, id := range folders {
		ip, port := discogsBridge.GetIP("recordcollection")
		conn, err2 := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
		if err2 != nil {
			return result, err2
		}

		if err2 == nil {
			defer conn.Close()
			client := pbrc.NewRecordCollectionServiceClient(conn)

			rel, err3 := client.GetRecords(ctx, &pbrc.GetRecordsRequest{Filter: &pbrc.Record{Release: &pbd.Release{FolderId: id}}})
			if err3 != nil {
				return result, err3
			}
			result = append(result, rel.GetRecords()...)
		}
	}

	utils.Trace(ctx, "getReleases", time.Now(), pbt.Milestone_END_FUNCTION, "recordsorganiser")
	return result, nil
}

func (discogsBridge prodBridge) getReleasesWithGoal(ctx context.Context, folders []int32) ([]*pbrc.Record, error) {
	ctx = utils.Trace(ctx, "getReleases", time.Now(), pbt.Milestone_START_FUNCTION, "recordsorganiser")
	var result []*pbrc.Record

	for _, id := range folders {
		ip, port := discogsBridge.GetIP("recordcollection")
		conn, err2 := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
		if err2 != nil {
			return result, err2
		}

		if err2 == nil {
			defer conn.Close()
			client := pbrc.NewRecordCollectionServiceClient(conn)

			rel, err3 := client.GetRecords(ctx, &pbrc.GetRecordsRequest{Filter: &pbrc.Record{Release: &pbd.Release{FolderId: id}}})
			if err3 != nil {
				return result, err3
			}
			result = append(result, rel.GetRecords()...)
		}
	}

	utils.Trace(ctx, "getReleases", time.Now(), pbt.Milestone_END_FUNCTION, "recordsorganiser")
	return result, nil
}

// DoRegister does RPC registration
func (s *Server) DoRegister(server *grpc.Server) {
	pb.RegisterOrganiserServiceServer(server, s)
}

// Mote promotes/demotes this server
func (s *Server) Mote(ctx context.Context, master bool) error {
	if master {
		err := s.readOrg(ctx)
		return err
	}

	return nil
}

// GetState gets the state of the server
func (s Server) GetState() []*pbgs.State {
	return []*pbgs.State{
		&pbgs.State{Key: "OrgTime", Text: fmt.Sprintf("%v", s.lastOrgTime)},
		&pbgs.State{Key: "OrgFold", Text: s.lastOrgFolder},
		&pbgs.State{Key: "sort_map_size", Value: int64(len(s.sortMap))},
	}
}

// InitServer builds an initial server
func InitServer() *Server {
	server := &Server{&goserver.GoServer{}, prodBridge{}, &pb.Organisation{}, &prodGh{}, time.Second, "", make(map[int32]*pb.SortMapping)}
	server.PrepServer()
	server.bridge = &prodBridge{Resolver: server.GetIP}
	server.GoServer.KSclient = *keystoreclient.GetClient(server.GetIP)

	server.Register = server

	return server
}

// ReportHealth alerts if we're not healthy
func (s Server) ReportHealth() bool {
	return true
}

func (s *Server) checkOrg(ctx context.Context) {
	for _, loc := range s.org.GetLocations() {
		if loc.ReorgTime == 0 {
			s.RaiseIssue(ctx, "Add reorg time", fmt.Sprintf("Add a reorg time span for %v", loc.GetName()), false)
		} else if loc.ReorgTime > 0 {
			cTime := int64(time.Now().Sub(time.Unix(loc.LastReorg, 0)).Seconds())
			if cTime > loc.ReorgTime {
				s.RaiseIssue(ctx, "Reorg", fmt.Sprintf("Please reorg %v", loc.GetName()), false)
			}
		}
	}
}

func main() {
	var quiet = flag.Bool("quiet", true, "Show log output")
	flag.Parse()

	if *quiet {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}
	log.Printf("Logging is on!")

	server := InitServer()

	server.GoServer.Killme = true
	server.RegisterServer("recordsorganiser", false)
	server.RegisterRepeatingTask(server.checkQuota, "check_quota", time.Hour)
	server.RegisterRepeatingTask(server.checkOrg, "check_org", time.Hour)

	server.Log(fmt.Sprintf("Starting %v", server.GoServer.RunningFile))
	server.Serve()
}
