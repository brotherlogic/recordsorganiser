package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/brotherlogic/goserver"
	"github.com/brotherlogic/goserver/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pbgs "github.com/brotherlogic/goserver/proto"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	rcpb "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
)

// Server the configuration for the syncer
type Server struct {
	*goserver.GoServer
	bridge discogsBridge
}

type discogsBridge interface {
	getReleases(ctx context.Context, folders []int32) ([]int32, error)
	getRecord(ctx context.Context, instanceID int32) (*pbrc.Record, error)
	updateRecord(ctx context.Context, req *pbrc.UpdateRecordRequest) (*pbrc.UpdateRecordsResponse, error)
}

func convert(exs []*pb.LabelExtractor) map[int32]string {
	m := make(map[int32]string)
	for _, ex := range exs {
		m[ex.LabelId] = ex.Extractor
	}
	return m
}

func (s *Server) markOverQuota(ctx context.Context, c *pb.Location) error {
	if c.GetQuota().GetAbsoluteWidth() > 0 {
		return s.processAbsoluteWidthQuota(ctx, c)
	}

	if c.GetQuota().GetTotalWidth() > 0 {
		return s.processWidthQuota(ctx, c)
	}
	return s.processQuota(ctx, c)
}

var (
	sizes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "recordsorganiser_in_box",
		Help: "Various Wait Times",
	}, []string{"location"})

	swidths = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "recordsorgainser_slot_widths",
		Help: "Widthof slots",
	}, []string{"location", "slot"})

	twidth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "recordsorganiser_total_width",
		Help: "Widthof slots",
	}, []string{"location", "filed"})

	tcount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "recordsorganiser_total_count",
		Help: "Widthof slots",
	}, []string{"location"})

	fwidth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "recordsorganiser_folder_width",
		Help: "Widthof slots",
	}, []string{"folder"})

	awidth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "recordsorganiser_average_width",
		Help: "Widthof slots",
	}, []string{"location"})
	otime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "recordsorganiser_org_time",
		Help: "Time take to organise a slot",
	}, []string{"location"})
	align = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "recordsorganiser_align_probs",
		Help: "Time take to organise a slot",
	}, []string{"location"})
)

func (s *Server) organiseLocation(ctx context.Context, cache *pb.SortingCache, c *pb.Location, org *pb.Organisation) (int32, error) {
	t := time.Now()
	defer func() {
		otime.With(prometheus.Labels{"location": c.GetName()}).Set(float64(time.Since(t).Milliseconds()))
	}()

	var noverall []*pbrc.Record
	var gaps []int
	widths := make(map[int32]float64)
	fwidths := []float64{1}
	tw := make(map[int32]string)
	fw := make(map[int32]int32)
	maxorder := int32(0)
	for _, ord := range c.GetFolderOrder() {
		if ord > maxorder {
			maxorder = ord
		}
	}
	for order := int32(0); order <= maxorder; order++ {
		var lfold []int32
		var sorter pb.Location_Sorting
		fg := false
		for key, val := range c.GetFolderOrder() {
			if val == order {
				lfold = append(lfold, key)
				sorter = c.GetFolderSort()[key]
				if c.GetHardGap()[key] {
					fg = true
				}
			}
		}
		if fg {
			gaps = append(gaps, len(noverall))
		}

		ids, err := s.bridge.getReleases(ctx, lfold)
		if err != nil {
			return -1, err
		}

		tfr := []*pbrc.Record{}
		tfr2 := []int32{}
		for _, id := range ids {
			if getEntry(cache, id) == nil {
				r, err := s.bridge.getRecord(ctx, id)
				if err != nil {
					return -1, err
				}
				appendCache(cache, r)
			}

			r, err := s.bridge.getRecord(ctx, id)
			if status.Convert(err).Code() != codes.OutOfRange {
				if err != nil {
					return -1, err
				}

				entry := appendCache(cache, r)
				//widths[r.GetRelease().GetInstanceId()] = float64(r.GetMetadata().GetRecordWidth())
				widths[id] = entry.GetWidth()

				if widths[id] > 0 {
					fwidths = append(fwidths, widths[id])
				}

				//tw[id] = r.GetMetadata().GetFiledUnder().String()
				tw[id] = entry.GetFilled()
				//fw[id] = r.GetRelease().GetFolderId()
				fw[id] = entry.GetFolder()

				tfr = append(tfr, r)
				tfr2 = append(tfr2, id)
			}
		}

		switch sorter {
		case pb.Location_BY_RELEASE_DATE:
			sort.Sort(ByEarliestReleaseDate(tfr))
		case pb.Location_BY_DATE_ADDED:
			sort.Sort(ByDateAdded(tfr))
		case pb.Location_BY_LABEL_CATNO:
			sort.Sort(ByLabelCat{tfr, convert(org.GetExtractors()), s.CtxLog, cache})
			sort.Sort(ByCachedLabelCat{tfr2, cache})

			issue := false
			count := 0
			for i := range tfr {
				if tfr[i].GetRelease().GetInstanceId() != tfr2[i] {
					align.With(prometheus.Labels{"location": c.GetName()}).Inc()
					count++
					counts1 := "Aligning on " + c.GetName() + "\n"
					if i > 0 {
						counts1 += fmt.Sprintf("%v vs %v\n", tfr[i-1].GetRelease().GetInstanceId(), tfr2[i-1])
					}
					counts1 += fmt.Sprintf("%v vs %v\n", tfr[i].GetRelease().GetInstanceId(), tfr2[i])
					if i+1 < len(tfr) {
						counts1 += fmt.Sprintf("%v vs %v\n", tfr[i+1].GetRelease().GetInstanceId(), tfr2[i+1])
					}

					// Tele music has two records with the same catalogue number and we don't care about Sold
					if !issue && tfr2[i] != 870564607 && tfr2[i] != 635886064 && c.GetName() != "Sold" {
						if c.GetName() != "Limbo" {
							s.RaiseIssue("Alignment Issue", counts1)
							issue = true
						}
					}
				}
			}
		case pb.Location_BY_FOLDER_THEN_DATE:
			sort.Sort(ByFolderThenRelease(tfr))
		case pb.Location_BY_MOVE_TIME:
			sort.Sort(ByDateMoved(tfr))
		case pb.Location_BY_LAST_LISTEN:
			sort.Sort(ByLastListen(tfr))
		}

		noverall = append(noverall, tfr...)
	}

	sort.Float64s(fwidths)
	mslot := make(map[int32]int)
	for _, slot := range c.GetFolderIds() {
		mslot[slot] = 999
	}

	//Before splitting let the org group records
	overall := noverall
	var mapper map[int32][]*rcpb.Record
	if c.CombineSimilar {
		overall, mapper = s.collapse(ctx, noverall, cache)
	}

	awidth.With(prometheus.Labels{"location": c.GetName()}).Set(float64(fwidths[len(fwidths)/2]))
	records := s.Split(ctx, c.GetName(), overall, float32(c.GetSlots()), float32(c.GetQuota().GetTotalWidth()), gaps, c.GetAllowAdjust(), fwidths[len(fwidths)/2])

	c.ReleasesLocation = []*pb.ReleasePlacement{}
	for slot, recs := range records {
		total := float32(0)
		for i, rinloc := range expand(recs, mapper) {
			sl := slot + 1
			if sl < mslot[rinloc.GetRelease().GetFolderId()] {
				mslot[rinloc.GetRelease().GetFolderId()] = sl
			}
			c.ReleasesLocation = append(c.ReleasesLocation,
				&pb.ReleasePlacement{
					Slot:            int32(slot + 1),
					Index:           int32(i),
					InstanceId:      rinloc.GetRelease().InstanceId,
					Title:           rinloc.GetRelease().Title,
					DeterminedWidth: getFormatWidth(rinloc, fwidths[len(fwidths)/2])})
			total += getFormatWidth(rinloc, fwidths[len(fwidths)/2])
		}

	}

	tcount.With(prometheus.Labels{"location": c.GetName()}).Set(float64(len(c.ReleasesLocation)))

	for folder, mi := range mslot {
		fstart.With(prometheus.Labels{"location": c.GetName(), "folder": fmt.Sprintf("%v", folder)}).Set(float64(mi))
	}

	//Make any quota adjustments - we only do width ajdustments
	if c.GetQuota().GetAbsoluteWidth() > 0 {
		s.markOverQuota(ctx, c)
	}

	slotWidths := make(map[int]float64)
	twf := make(map[string]float64)
	fwf := make(map[int32]float64)
	for _, ent := range c.GetReleasesLocation() {
		slotWidths[int(ent.GetSlot())] += float64(ent.GetDeterminedWidth())
		twf[tw[ent.GetInstanceId()]] += float64(ent.GetDeterminedWidth())
		fwf[fw[ent.GetInstanceId()]] += float64(ent.GetDeterminedWidth())
	}

	for slot, width := range slotWidths {
		swidths.With(prometheus.Labels{"location": c.GetName(), "slot": fmt.Sprintf("%v", slot)}).Set(width)

	}

	for key, val := range twf {
		twidth.With(prometheus.Labels{"location": c.GetName(), "filed": key}).Set(val)
	}

	for key, val := range fwf {
		fwidth.With(prometheus.Labels{"folder": fmt.Sprintf("%v", key)}).Set(val)
	}

	s.saveCache(ctx, cache)
	return int32(len(overall)), s.saveOrg(ctx, org)
}

// Bridge that accesses discogs syncer server
type prodBridge struct {
	dial func(ctx context.Context, server string) (*grpc.ClientConn, error)
	log  func(context.Context, string)
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

		defer conn.Close()
		client := pbrc.NewRecordCollectionServiceClient(conn)

		rel, err3 := client.QueryRecords(ctx, &pbrc.QueryRecordsRequest{Query: &pbrc.QueryRecordsRequest_FolderId{FolderId: id}})
		if err3 != nil {
			return result, err3
		}
		result = append(result, rel.GetInstanceIds()...)
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

	return server
}

// ReportHealth alerts if we're not healthy
func (s Server) ReportHealth() bool {
	return true
}

func main() {
	server := InitServer()
	server.PrepServer("recordsorganiser")

	server.bridge = &prodBridge{dial: server.FDialServer, log: server.CtxLog}
	server.Register = server

	err := server.RegisterServerV2(false)
	if err != nil {
		return
	}

	go func() {
		ctx, cancel := utils.ManualContext("recorginit", time.Minute*10)
		err = server.metrics(ctx)
		cancel()
		if err != nil {
			return
		}
	}()

	server.Serve()
}
