package main

import (
	"flag"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/brotherlogic/goserver"
	"github.com/brotherlogic/keystore/client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pbs "github.com/brotherlogic/discogssyncer/server"
	pbdi "github.com/brotherlogic/discovery/proto"
	pbd "github.com/brotherlogic/godiscogs"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

const (
	//CurrKey the current state of the collection
	CurrKey = "/github.com/brotherlogic/recordsorganiser/curr"

	//PrevKey the old state of the collection
	PrevKey = "/github.com/brotherlogic/recordsorganiser/prev"
)

// Bridge that accesses discogs syncer server
type prodBridge struct{}

func (discogsBridge prodBridge) getMetadata(rel *pbd.Release) *pbs.ReleaseMetadata {
	ip, port := getIP("discogssyncer")
	conn, _ := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
	defer conn.Close()
	client := pbs.NewDiscogsServiceClient(conn)
	meta, _ := client.GetMetadata(context.Background(), rel)
	return meta
}

func (discogsBridge prodBridge) moveToFolder(move *pbs.ReleaseMove) {
	ip, port := getIP("discogssyncer")
	conn, _ := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
	defer conn.Close()
	client := pbs.NewDiscogsServiceClient(conn)
	client.MoveToFolder(context.Background(), move)
}

func (discogsBridge prodBridge) getReleases(folders []int32) []*pbd.Release {
	var result []*pbd.Release

	list := &pbs.FolderList{}
	for _, id := range folders {
		list.Folders = append(list.Folders, &pbd.Folder{Id: id})
	}

	ip, port := getIP("discogssyncer")
	conn, _ := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
	defer conn.Close()
	client := pbs.NewDiscogsServiceClient(conn)

	rel, _ := client.GetReleasesInFolder(context.Background(), list)
	result = rel.GetReleases()

	return result
}

func (discogsBridge prodBridge) getRelease(ID int32) *pbd.Release {
	var result *pbd.Release

	ip, port := getIP("discogssyncer")
	conn, _ := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
	defer conn.Close()
	client := pbs.NewDiscogsServiceClient(conn)

	rel, _ := client.GetCollection(context.Background(), &pbs.Empty{})
	for _, item := range rel.GetReleases() {
		if item.Id == ID {
			return item
		}
	}
	return result
}

func compare(collectionStart *pb.Organisation, collectionEnd *pb.Organisation) []*pb.LocationMove {
	var moves []*pb.LocationMove

	for _, folder := range collectionEnd.Locations {
		var matcher = &pb.Location{}
		for _, otherFolder := range collectionStart.Locations {
			if otherFolder.Name == folder.Name {
				matcher = otherFolder
			}
		}

		for i := 1; i <= int(folder.Units); i++ {
			diff := getMoves(matcher.ReleasesLocation, folder.ReleasesLocation, i, folder.Name)
			moves = append(moves, diff...)
		}
	}

	// Correct the slot moves
	for _, move := range moves {
		if !move.SlotMove {
			if move.New != nil {
				for _, location := range collectionStart.Locations {
					for _, placement := range location.ReleasesLocation {
						if placement.ReleaseId == move.New.ReleaseId {
							if location.Name == move.New.Folder {
								move.SlotMove = true
							}
						}
					}
				}
			} else {
				for _, location := range collectionStart.Locations {
					for _, placement := range location.ReleasesLocation {
						if placement.ReleaseId == move.Old.ReleaseId {
							if location.Name == move.Old.Folder {
								move.SlotMove = true
							}
						}
					}
				}
			}
		}
	}

	return moves
}

// DoRegister does RPC registration
func (s Server) DoRegister(server *grpc.Server) {
	pb.RegisterOrganiserServiceServer(server, &s)
}

// Mote promotes/demotes this server
func (s Server) Mote(master bool) error {
	return nil
}

func (s Server) save() {

	// Always update the timestamp on a save
	s.currOrg.Timestamp = time.Now().Unix()
	s.KSclient.Save(CurrKey, s.currOrg)
	s.KSclient.Save(PrevKey, s.pastOrg)
}

func (s Server) load(key string) (*pb.Organisation, error) {
	collection := &pb.Organisation{}
	data, err := s.KSclient.Read(key, collection)
	if err != nil {
		return nil, err
	}

	return data.(*pb.Organisation), nil
}

func (s *Server) loadLatest() error {
	curr, err := s.load(CurrKey)
	if err != nil {
		return err
	}
	past, err := s.load(PrevKey)
	if err != nil {
		return err
	}

	s.currOrg = curr
	s.pastOrg = past

	return nil
}

// InitServer builds an initial server
func InitServer() Server {
	server := Server{&goserver.GoServer{}, prodBridge{}, &pb.Organisation{}, &pb.Organisation{}}
	server.GoServer.KSclient = *keystoreclient.GetClient(getIP)
	server.loadLatest()
	server.Register = server

	return server
}

func getIP(servername string) (string, int) {
	conn, _ := grpc.Dial("192.168.86.64:50055", grpc.WithInsecure())
	defer conn.Close()

	registry := pbdi.NewDiscoveryServiceClient(conn)
	entry := pbdi.RegistryEntry{Name: servername}
	r, err := registry.Discover(context.Background(), &entry)

	if err != nil {
		log.Printf("Failed: %v", err)
		return "", -1
	}

	return r.Ip, int(r.Port)
}

func test() {
	dServer, dPort := getIP("discogssyncer")

	dConn, _ := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	defer dConn.Close()
	dClient := pbs.NewDiscogsServiceClient(dConn)

	list := &pbs.FolderList{}
	list.Folders = append(list.Folders, &pbd.Folder{Name: "12s"})

	releases, err := dClient.GetReleasesInFolder(context.Background(), list)

	if err != nil {
		log.Fatal(err)
	}

	sort.Sort(pbd.ByLabelCat(releases.Releases))
	splits := pbd.Split(releases.Releases, 8)
	count := 1
	for _, split := range splits[0] {
		log.Printf("%v - %v", count, split.Title)
		count++
	}
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
	log.Printf("Logging is on!")

	server := InitServer()

	server.PrepServer()
	server.GoServer.Killme = true
	server.RegisterServer("recordsorganiser", false)
	log.Printf("Collection size: %v", len(server.currOrg.Locations))
	server.Serve()
}
