package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/brotherlogic/goserver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pbgs "github.com/brotherlogic/goserver/proto"
	"github.com/brotherlogic/goserver/utils"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	rcpb "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

// Bridge that accesses discogs syncer server
type prodBridge struct {
	dial func(ctx context.Context, server string) (*grpc.ClientConn, error)
	log  func(string)
}

var (
	count = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "recordsorganiser_cache_count",
		Help: "The size of the tracking queue",
	})
	size = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "recordsorganiser_cache_size",
		Help: "The size of the tracking queue",
	})
)

const (
	//KEY is where we store the org
	KEY = "github.com/brotherlogic/recordsorganiser/org"

	CACHE_KEY = "github.com/brotherlgoic/recordsorganiser/cache"
)

func (s *Server) loadCache(ctx context.Context) (*pb.SortingCache, error) {
	data, err := s.LoadData(ctx, CACHE_KEY, 0.5)
	if err != nil {
		if status.Convert(err).Code() == codes.InvalidArgument {
			return &pb.SortingCache{}, nil
		}
		return nil, err
	}

	cache := &pb.SortingCache{}
	err = proto.Unmarshal(data, cache)
	if err != nil {
		return nil, err
	}

	count.Set(float64(len(cache.GetCache())))
	size.Set(float64(proto.Size(cache)))

	return cache, nil
}

func (s *Server) saveCache(ctx context.Context, config *pb.SortingCache) error {
	data, err := proto.Marshal(config)
	if err != nil {
		return err
	}
	return s.SaveData(ctx, data, CACHE_KEY, 0.5)
}

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

		if location.GetFolderOrder() == nil {
			location.FolderOrder = make(map[int32]int32)
			location.FolderSort = make(map[int32]pb.Location_Sorting)

			for _, folder := range location.GetFolderIds() {
				location.FolderOrder[folder] = 0
				location.FolderSort[folder] = location.Sort
			}
		}

		seen := make(map[int32]bool)
		var done []int32
		for _, val := range location.GetFolderIds() {
			if !seen[val] {
				done = append(done, val)
				seen[val] = true
			}
		}
		location.FolderIds = done

		if location.GetName() == "Holding" {
			location.FolderIds = []int32{3578980}
			delete(location.FolderOrder, int32(673768))
			delete(location.FolderSort, int32(673768))
		}

		if location.GetName() == "12 Inches" {
			location.CombineSimilar = true
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
		&pbgs.State{Key: "yes", Value: int64(123467)},
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

	ctx, cancel := utils.ManualContext("recorginit", time.Minute*10)
	err = server.metrics(ctx)
	cancel()
	if err != nil {
		return
	}

	server.Serve()
}
