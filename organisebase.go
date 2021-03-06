package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/brotherlogic/goserver"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pbgs "github.com/brotherlogic/goserver/proto"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	rcpb "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

// Bridge that accesses discogs syncer server
type prodBridge struct {
	dial func(ctx context.Context, server string) (*grpc.ClientConn, error)
	log  func(string)
}

const (
	//KEY is where we store the org
	KEY = "github.com/brotherlogic/recordsorganiser/org"
)

func (s *Server) readOrg(ctx context.Context) (*pb.Organisation, error) {
	org := &pb.Organisation{}
	data, _, err := s.KSclient.Read(ctx, KEY, org)

	if err != nil {
		return nil, err
	}
	org = data.(*pb.Organisation)

	// Verify that all locations have their play settings set
	locations := []string{}
	for _, location := range org.Locations {
		if location.InPlay == pb.Location_PLAY_UNKNOWN {
			locations = append(locations, location.Name)
		}
	}

	if len(locations) > 0 {
		s.RaiseIssue("Missing Play State", fmt.Sprintf("These locations: %v are missing play states", locations))
	}

	return org, nil
}

func (s *Server) saveOrg(ctx context.Context, org *pb.Organisation) error {
	return s.KSclient.Save(ctx, KEY, org)
}

func (discogsBridge prodBridge) getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error) {
	conn, err := discogsBridge.dial(ctx, "recordcollection")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := pbrc.NewRecordCollectionServiceClient(conn)

	rec, err := client.GetRecord(ctx, &pbrc.GetRecordRequest{InstanceId: instanceID})
	if err != nil {
		return nil, err
	}

	return rec.GetRecord(), err
}
func (discogsBridge prodBridge) updateRecord(ctx context.Context, update *pbrc.UpdateRecordRequest) (*pbrc.UpdateRecordsResponse, error) {
	conn, err2 := discogsBridge.dial(ctx, "recordcollection")
	if err2 == nil {
		defer conn.Close()
		client := pbrc.NewRecordCollectionServiceClient(conn)
		return client.UpdateRecord(ctx, update)
	}

	return nil, fmt.Errorf("Unable to dial recordcollection: %v", err2)
}

func (discogsBridge prodBridge) getReleases(ctx context.Context, folders []int32) ([]int32, error) {
	var result []int32

	for _, id := range folders {
		conn, err2 := discogsBridge.dial(ctx, "recordcollection")
		if err2 != nil {
			return result, err2
		}

		if err2 == nil {
			defer conn.Close()
			client := pbrc.NewRecordCollectionServiceClient(conn)

			rel, err3 := client.QueryRecords(ctx, &pbrc.QueryRecordsRequest{Query: &pbrc.QueryRecordsRequest_FolderId{id}})
			if err3 != nil {
				return result, err3
			}
			result = append(result, rel.GetInstanceIds()...)
		}
	}

	return result, nil
}

// DoRegister does RPC registration
func (s *Server) DoRegister(server *grpc.Server) {
	pb.RegisterOrganiserServiceServer(server, s)
	rcpb.RegisterClientUpdateServiceServer(server, s)
}

// Shutdown the server
func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}

// Mote promotes/demotes this server
func (s *Server) Mote(ctx context.Context, master bool) error {
	return nil
}

// GetState gets the state of the server
func (s Server) GetState() []*pbgs.State {
	return []*pbgs.State{
		&pbgs.State{Key: "yes", Value: int64(12346)},
	}
}

// InitServer builds an initial server
func InitServer() *Server {
	server := &Server{
		&goserver.GoServer{},
		prodBridge{},
	}
	server.PrepServer()

	server.bridge = &prodBridge{dial: server.FDialServer, log: server.Log}
	server.Register = server

	return server
}

// ReportHealth alerts if we're not healthy
func (s Server) ReportHealth() bool {
	return true
}

func (s *Server) checkbOrg(ctx context.Context, org *pb.Organisation) error {
	for _, loc := range org.GetLocations() {
		if loc.ReorgTime == 0 {
			s.RaiseIssue("Add reorg time", fmt.Sprintf("Add a reorg time span for %v", loc.GetName()))
		} else if loc.ReorgTime > 0 {
			cTime := int64(time.Now().Sub(time.Unix(loc.LastReorg, 0)).Seconds())
			if cTime > loc.ReorgTime {
				s.RaiseIssue("Reorg", fmt.Sprintf("Please reorg %v", loc.GetName()))
			}
		}
	}
	return nil
}

func main() {
	var quiet = flag.Bool("quiet", true, "Show log output")
	flag.Parse()

	if *quiet {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}

	server := InitServer()

	err := server.RegisterServerV2("recordsorganiser", false, true)
	if err != nil {
		return
	}

	server.Serve()
}
