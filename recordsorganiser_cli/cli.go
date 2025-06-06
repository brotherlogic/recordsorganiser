package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/brotherlogic/goserver/utils"
	"google.golang.org/grpc"

	pbgd "github.com/brotherlogic/godiscogs/proto"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	"github.com/brotherlogic/recordsorganiser/locator"
	pb "github.com/brotherlogic/recordsorganiser/proto"

	//Needed to pull in gzip encoding init
	_ "google.golang.org/grpc/encoding/gzip"
)

func isTenInch(release *pbgd.Release) bool {
	for _, format := range release.GetFormats() {
		if format.GetName() == "10\"" {
			return true
		}
		for _, desc := range format.GetDescriptions() {
			if desc == "10\"" {
				return true
			}
		}
	}
	return false
}

// ByReleaseDate sorts by the given release date
// but puts matched records up front and keepers in the rear
type ByReleaseDate []*pbrc.Record

func (a ByReleaseDate) Len() int      { return len(a) }
func (a ByReleaseDate) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByReleaseDate) Less(i, j int) bool {
	if a[i].GetMetadata().Keep != a[j].GetMetadata().Keep {
		if a[i].GetMetadata().Keep == pbrc.ReleaseMetadata_KEEPER {
			return false
		}
		if a[j].GetMetadata().Keep == pbrc.ReleaseMetadata_KEEPER {
			return true
		}
	}
	if a[i].GetRelease().Released != a[j].GetRelease().Released {
		return a[i].GetRelease().Released > a[j].GetRelease().Released
	}
	return strings.Compare(a[i].GetRelease().Title, a[j].GetRelease().Title) < 0
}

func locateRelease(ctx context.Context, c pb.OrganiserServiceClient, id int32) {
	conn, err := utils.LFDialServer(ctx, "recordcollection")

	if err != nil {
		log.Fatalf("Unable to dial: %v", err)
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	ids, err := client.QueryRecords(ctx, &pbrc.QueryRecordsRequest{Query: &pbrc.QueryRecordsRequest_ReleaseId{int32(id)}})
	if err != nil {
		log.Fatalf("Unable to get record %v -> %v", id, err)
	}

	if len(ids.GetInstanceIds()) == 0 {
		fmt.Printf("No records with that id\n")
	}

	for _, id := range ids.GetInstanceIds() {
		location, err := c.Locate(ctx, &pb.LocateRequest{InstanceId: id})
		if err != nil {
			fmt.Printf("Unable to locate instance (%v) of %v because %v\n", id, id, err)
		} else {
			rec, err := client.GetRecord(ctx, &pbrc.GetRecordRequest{InstanceId: id})
			if err != nil {
				fmt.Printf("Error getting record: %v", err)
				return
			}
			fmt.Printf("%v (%v) is in %v\n", rec.GetRecord().GetRelease().Title, rec.GetRecord().GetRelease().InstanceId, location.GetFoundLocation().GetName())

			for i, r := range location.GetFoundLocation().GetReleasesLocation() {
				if r.GetInstanceId() == id {
					fmt.Printf("Slot %v\n", r.GetSlot())
					if i > 0 {
						fmt.Printf("%v. %v\n", i-1, getReleaseString(ctx, location.GetFoundLocation().GetReleasesLocation()[i-1], false, false))
					}
					fmt.Printf("%v. %v\n", i, getReleaseString(ctx, location.GetFoundLocation().GetReleasesLocation()[i], false, false))
					fmt.Printf("%v. %v\n", i+1, getReleaseString(ctx, location.GetFoundLocation().GetReleasesLocation()[i+1], false, false))
				}
			}
		}
	}
}

func getReleaseString(ctx context.Context, loc *pb.ReleasePlacement, showSleeve, simple bool) string {
	rec, err := getRecord(ctx, loc.InstanceId)
	if err != nil {
		return fmt.Sprintf("%v", err)
	}
	/*if rec.GetMetadata().GetFiledUnder() == pbrc.ReleaseMetadata_FILE_DIGITAL || rec.GetMetadata().GetFiledUnder() == pbrc.ReleaseMetadata_FILE_CD {
		return ""
	}*/

	if simple {
		return rec.GetRelease().GetTitle()
	}

	sleeve := ""
	if showSleeve {
		sleeve = fmt.Sprintf("%v", rec.GetMetadata().GetSleeve())
	}
	return fmt.Sprintf("%v (%v) %v ", rec.GetRelease().GetId(), rec.GetMetadata().GetRecordWidth(), rec.GetMetadata().GetLastListenTime()) + loc.Title + " " + fmt.Sprintf("%v", rec.GetMetadata().GetFiledUnder()) + " [" + strconv.Itoa(int(loc.InstanceId)) + "] - " + fmt.Sprintf("%v", rec.GetMetadata().GetCategory()) + " {" + fmt.Sprintf("%v", loc.GetDeterminedWidth()) + "} + " + fmt.Sprintf("%v", rec.GetMetadata().GetLastMoveTime()) + " [" + fmt.Sprintf("%v", rec.GetRelease().GetLabels()) + "]" + sleeve + fmt.Sprintf(" score%v", rec.Metadata.GetSetRating())
}

func getRecord(ctx context.Context, id int32) (*pbrc.Record, error) {
	conn, err := utils.LFDialServer(ctx, "recordcollection")

	if err != nil {
		log.Fatalf("Unable to dial: %v", err)
	}
	defer conn.Close()

	client := pbrc.NewRecordCollectionServiceClient(conn)
	val, err := client.GetRecord(ctx, &pbrc.GetRecordRequest{InstanceId: id})
	if err != nil {
		return nil, err
	}
	return val.GetRecord(), err
}

func isTwelve(ctx context.Context, instanceID int32) bool {
	conn, err := utils.LFDialServer(ctx, "recordcollection")
	defer conn.Close()

	if err != nil {
		log.Fatalf("Unable to dial: %v", err)
	}

	client := pbrc.NewRecordCollectionServiceClient(conn)
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	rel, err := client.GetRecord(ctx, &pbrc.GetRecordRequest{InstanceId: instanceID})
	if err != nil {
		log.Fatalf("unable to get record (%v): %v", instanceID, err)
	}

	if rel.GetRecord().GetMetadata().GetGoalFolder() == 1782105 {
		return false
	}

	for _, f := range rel.GetRecord().GetRelease().Formats {
		if f.Name == "LP" || f.Name == "Vinyl" {
			return true
		}
	}

	return false
}

func isSeven(ctx context.Context, instanceID int32) bool {
	conn, err := utils.LFDialServer(ctx, "recordcollection")
	defer conn.Close()

	if err != nil {
		log.Fatalf("Unable to dial: %v", err)
	}

	client := pbrc.NewRecordCollectionServiceClient(conn)
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	rel, err := client.GetRecord(ctx, &pbrc.GetRecordRequest{InstanceId: instanceID})
	if err != nil {
		log.Fatalf("unable to get record (%v): %v", instanceID, err)
	}

	if rel.GetRecord().GetMetadata().GetGoalFolder() == 1782105 {
		return false
	}

	for _, f := range rel.GetRecord().GetRelease().Formats {
		for _, d := range f.GetDescriptions() {
			if d == "7\"" {
				return true
			}
		}
	}

	return false
}

func get(ctx context.Context, client pb.OrganiserServiceClient, name string, force bool, slot int32, twelves bool, reset bool, sevens, showSleeve bool, justTen bool, simple bool) {
	locs, err := client.GetOrganisation(ctx, &pb.GetOrganisationRequest{OrgReset: reset, ForceReorg: force, Locations: []*pb.Location{&pb.Location{Name: name}}})
	if err != nil {
		log.Fatalf("Error reading locations: %v", err)
	}

	lastSlot := int32(1)
	total := float32(0)
	twidth := float32(0)
	tcount := float32(0)
	var widths []float64
	for _, loc := range locs.GetLocations() {
		fmt.Printf("%v (%v) -> %v [%v] with %v (%v) %v [Last reorg: %v from %v] (%v)\n", loc.GetName(), len(loc.GetReleasesLocation()), loc.GetFolderIds(), loc.GetQuota(), loc.Sort.String(), loc.GetSlots(), loc.GetSpillFolder(), time.Unix(loc.LastReorg, 0), loc.ReorgTime, loc.InPlay)
		fmt.Printf("%v\n", loc.GetFolderOrder())
		fmt.Printf("%v\n", loc.GetFolderSort())
		fmt.Printf("%v\n", loc.GetHardGap())
		fmt.Printf("%v\n", loc.GetQuota())

		for j, rloc := range loc.GetReleasesLocation() {
			if slot < 0 || rloc.GetSlot() == slot {
				if (!twelves || isTwelve(ctx, rloc.GetInstanceId())) && (!sevens || isSeven(ctx, rloc.GetInstanceId())) {
					if rloc.GetSlot() > lastSlot {
						fmt.Printf("\n")
						lastSlot = rloc.GetSlot()
					}
					rec, err := getRecord(ctx, rloc.InstanceId)
					if err != nil {
						log.Fatalf("ARFH: %v", err)
					}

					if !justTen || isTenInch(rec.GetRelease()) {

						fmt.Printf("%v [%v] %v %v [%v] \n", j, rloc.GetSlot(), rloc.GetInstanceId(), getReleaseString(ctx, rloc, showSleeve, simple), total)
						total += rloc.GetDeterminedWidth()
						if rec.GetMetadata().GetRecordWidth() > 0 {
							twidth += rec.GetMetadata().GetRecordWidth()
							tcount++
							widths = append(widths, float64(rec.Metadata.GetRecordWidth()))
						}
					}
				}
			}
		}
	}
	sort.Float64s(widths)

	fmt.Printf("Total Width: %v\n", twidth)

	if len(locs.GetLocations()) == 0 {
		fmt.Printf("No Locations Found!\n")
	}
}
func getFormatWidth(r *pbrc.Record) float32 {
	// Use the spine width if we have it
	if r.GetMetadata().GetRecordWidth() > 0 {
		return r.GetMetadata().GetRecordWidth()
	}

	// Death Waltz release are thicker than average
	for _, label := range r.GetRelease().GetLabels() {
		if label.Name == "Death Waltz Recording Company" || label.Name == "Now-Again Records" {
			return 12.2
		}
	}
	for _, format := range r.GetRelease().GetFormats() {
		if strings.Contains(format.Text, "Gatefold") {
			return 8.7
		}
		if strings.Contains(format.Text, "Box") {
			return 18.0
		}
	}

	//Regular record size
	return 3.7
}

func list(ctx context.Context, client pb.OrganiserServiceClient) {
	locs, err := client.GetOrganisation(ctx, &pb.GetOrganisationRequest{Locations: []*pb.Location{&pb.Location{}}})
	if err != nil {
		log.Fatalf("Error reading locations: %v", err)
	}

	for i, loc := range locs.GetLocations() {
		fmt.Printf("%v. %v [%v] (%v)\n", i, loc.GetName(), loc.GetInPlay(), loc.GetFolderIds())
	}

	if len(locs.GetLocations()) == 0 {
		fmt.Printf("No Locations Found!\n")
	}
}

func add(ctx context.Context, client pb.OrganiserServiceClient, name string, folders []int32, slots int32) {
	loc, err := client.AddLocation(ctx, &pb.AddLocationRequest{Add: &pb.Location{Name: name, FolderIds: folders, Slots: slots}})

	if err != nil {
		log.Fatalf("Error adding location: %v", err)
	}

	fmt.Printf("Added location: %v\n", len(loc.GetNow().GetLocations()))
}

func main() {
	ctx, cancel := utils.BuildContext("OrgCLI-"+os.Args[1], "recordsorganiser")
	defer cancel()

	conn, err := utils.LFDialServer(ctx, "recordsorganiser")
	if err != nil {
		log.Fatalf("Unable to dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewOrganiserServiceClient(conn)

	switch os.Args[1] {
	case "ping":
		client2 := pbrc.NewClientUpdateServiceClient(conn)
		ilocateFlags := flag.NewFlagSet("ILocate", flag.ExitOnError)
		var id = ilocateFlags.Int("id", -1, "The name of the location")

		if err := ilocateFlags.Parse(os.Args[2:]); err == nil {
			location, err := client2.ClientUpdate(ctx, &pbrc.ClientUpdateRequest{InstanceId: int32(*id)})
			fmt.Printf("%v -> %v\n", location, err)
		}
	case "cache":
		ilocateFlags := flag.NewFlagSet("ILocate", flag.ExitOnError)
		var id = ilocateFlags.Int("id", -1, "The name of the location")

		if err := ilocateFlags.Parse(os.Args[2:]); err == nil {
			cache, err := client.GetCache(ctx, &pb.GetCacheRequest{})
			if err != nil {
				log.Fatalf("Cannot read cache: %v", err)
			}
			for _, entry := range cache.GetCache().GetCache() {
				if entry.GetInstanceId() == int32(*id) {
					fmt.Printf("%v\n", entry)
				}
			}
		}
	case "ilocate":
		ilocateFlags := flag.NewFlagSet("ILocate", flag.ExitOnError)
		var id = ilocateFlags.Int("id", -1, "The name of the location")

		if err := ilocateFlags.Parse(os.Args[2:]); err == nil {
			location, err := locator.ReadableLocation(ctx, utils.LFDialServer, int32(*id), true)
			fmt.Printf("%v -> %v\n", location, err)
		}
	case "cupdate":
		val, _ := strconv.ParseInt(os.Args[2], 10, 32)
		client := pbrc.NewClientUpdateServiceClient(conn)
		resp, err := client.ClientUpdate(ctx, &pbrc.ClientUpdateRequest{InstanceId: int32(val)})

		fmt.Printf("%v and %v\n", resp, err)
	case "list":
		list(ctx, client)
	case "get":
		getLocationFlags := flag.NewFlagSet("GetLocation", flag.ExitOnError)
		var name = getLocationFlags.String("name", "", "The name of the location")
		var force = getLocationFlags.Bool("force", false, "Force a reorg")
		var slot = getLocationFlags.Int("slot", 1, "Slot to view")
		var twelves = getLocationFlags.Bool("twelves", false, "Just 12 inches")
		var sevens = getLocationFlags.Bool("sevens", false, "Just 7 inches")
		var reorg = getLocationFlags.Bool("reorg", false, "Do a full reorg")
		var showSleeve = getLocationFlags.Bool("sleeves", false, "Show sleeve state")
		var justTens = getLocationFlags.Bool("just_tens", false, "Just ten inches")
		var simple = getLocationFlags.Bool("simple", false, "Simple display")

		if err := getLocationFlags.Parse(os.Args[2:]); err == nil {
			get(ctx, client, *name, *force, int32(*slot), *twelves, *reorg, *sevens, *showSleeve, *justTens, *simple)
		}
	case "add":
		addLocationFlags := flag.NewFlagSet("AddLocation", flag.ExitOnError)
		var name = addLocationFlags.String("name", "", "The name of the new location")
		var slots = addLocationFlags.Int("slots", 0, "The number of slots in the location")
		var folderIds = addLocationFlags.String("folders", "", "The list of folder IDs")

		if err := addLocationFlags.Parse(os.Args[2:]); err == nil {
			nums := make([]int32, 0)
			for _, folderID := range strings.Split(*folderIds, ",") {
				v, err := strconv.ParseInt(folderID, 10, 32)
				if err != nil {
					log.Fatalf("Cannot parse folderid: %v", err)
				}
				nums = append(nums, int32(v))
			}
			add(ctx, client, *name, nums, int32(*slots))
		}
	case "locate":
		locateFlags := flag.NewFlagSet("Locate", flag.ExitOnError)
		var id = locateFlags.Int("id", -1, "The id of the release")
		if err := locateFlags.Parse(os.Args[2:]); err == nil {
			locateRelease(ctx, client, int32(*id))
		}
	case "quota":
		quotaFlags := flag.NewFlagSet("quota", flag.ExitOnError)
		var name = quotaFlags.String("name", "", "The name of the location to evaluate")
		var show = quotaFlags.Bool("show", false, "Show the over quota records")

		if err := quotaFlags.Parse(os.Args[2:]); err == nil {
			loc, err := client.GetOrganisation(ctx, &pb.GetOrganisationRequest{Locations: []*pb.Location{&pb.Location{Name: *name}}})

			if err != nil {
				log.Fatalf("Unable to get org: %v", err)
			}

			quot, err := client.GetQuota(ctx, &pb.QuotaRequest{FolderId: loc.GetLocations()[0].FolderIds[0], IncludeRecords: false})
			fmt.Printf("QUOTA = %v and %v\n", quot.GetOverQuota(), len(quot.InstanceId))
			if quot.GetOverQuota() && *show {
				fmt.Printf("%v is over quota by %v\n", *name, len(quot.InstanceId))
				for _, id := range quot.InstanceId {
					host, port, err := utils.Resolve("recordcollection", "recorg-what")
					if err != nil {
						log.Fatalf("Unable to reach collection: %v", err)
					}
					conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
					defer conn.Close()

					if err != nil {
						log.Fatalf("Unable to dial: %v", err)
					}

					client := pbrc.NewRecordCollectionServiceClient(conn)
					recs, err := client.GetRecord(ctx, &pbrc.GetRecordRequest{InstanceId: id})
					if err != nil {
						log.Fatalf("Unable to get record %v -> %v", id, err)
					}

					fmt.Printf("%v\n", recs.GetRecord().GetMetadata().Category)
				}
			}
		}
	case "sell":
		sellFlags := flag.NewFlagSet("sell", flag.ExitOnError)
		var name = sellFlags.String("name", "", "The name of the location to get")
		var assess = sellFlags.Bool("assess", false, "Auto assess the for sale records")
		var forcesell = sellFlags.Bool("force", false, "Auto assess the for sale records")
		var limit = sellFlags.Int("limit", -1, "Limit to include")

		if err := sellFlags.Parse(os.Args[2:]); err == nil {
			host, port, err := utils.Resolve("recordcollection", "record-sell")
			if err != nil {
				log.Fatalf("Unable to reach collection: %v", err)
			}
			conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
			defer conn.Close()

			if err != nil {
				log.Fatalf("Unable to dial: %v", err)
			}

			rclient := pbrc.NewRecordCollectionServiceClient(conn)

			loc, err := client.GetQuota(ctx, &pb.QuotaRequest{IncludeRecords: true, Name: *name})
			if err != nil {
				log.Fatalf("Error in get quota: %v", err)
			}

			records := make([]*pbrc.Record, 0)
			minScore := float64(6)
			for _, i := range loc.InstanceId {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()

				recs, err := rclient.GetRecord(ctx, &pbrc.GetRecordRequest{InstanceId: i})
				if err != nil {
					log.Fatalf("Error : %v", err)
				}

				r := recs.GetRecord()
				records = append(records, r)
				score := float64(r.GetRelease().Rating)
				if score == 0 {
					score = float64(r.GetMetadata().OverallScore)
				}
				if score > 0.0 && score < minScore {
					minScore = score
				}
			}

			if *limit > 0 && int(minScore) < *limit {
				minScore = float64(*limit)
			}

			fmt.Printf("Checking on %v records with the min score being %v\n", len(records), minScore)

			total := len(records) - int(loc.Quota.NumOfSlots) + 4
			count := 0

			//Sort by release date
			sort.Sort(ByReleaseDate(records))
			for _, r := range records {
				if count > total {
					break
				}
				score := float64(r.GetRelease().Rating)
				if score == 0 {
					score = float64(r.GetMetadata().OverallScore)
				}
				if score <= minScore {
					count++
					fmt.Printf("SELL: [%v] %v\n", r.GetRelease().InstanceId, r.GetRelease().Title)
					if *assess {
						up := &pbrc.UpdateRecordRequest{Update: &pbrc.Record{Release: &pbgd.Release{InstanceId: r.GetRelease().InstanceId}, Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_ASSESS}}}
						_, err = rclient.UpdateRecord(context.Background(), up)
						if err != nil {
							log.Fatalf("Error updating record: %v", err)
						}
					}
					if *forcesell {
						up := &pbrc.UpdateRecordRequest{Update: &pbrc.Record{Release: &pbgd.Release{InstanceId: r.GetRelease().InstanceId}, Metadata: &pbrc.ReleaseMetadata{Category: pbrc.ReleaseMetadata_PREPARE_TO_SELL}}}
						_, err = rclient.UpdateRecord(context.Background(), up)
						if err != nil {
							log.Fatalf("Error updating record: %v", err)
						}
					}

				}
			}
		}
	case "update":
		updateLocationFlags := flag.NewFlagSet("UpdateLocation", flag.ExitOnError)
		var name = updateLocationFlags.String("name", "", "The name of the new location")
		var folder = updateLocationFlags.Int("folder", 0, "The folder to add to the location")
		var quota = updateLocationFlags.Int("quota", 0, "The new quota to add to the location")
		var maxWidth = updateLocationFlags.Int("width", 0, "The max width per slot")
		var sort = updateLocationFlags.String("sort", "", "The new sorting mechanism")
		var alert = updateLocationFlags.Bool("alert", true, "Whether we should alert on this location")
		var spill = updateLocationFlags.Int("spill", 0, "The spill folder for this location")
		var slots = updateLocationFlags.Int("slots", 0, "The new number of slots for this location")
		var optOut = updateLocationFlags.Bool("out_out", false, "To opt this location out of quota alerts.")
		var reorgTime = updateLocationFlags.Int("reorg", 0, "The time needed to do a full reorg.")
		var delete = updateLocationFlags.Bool("delete", false, "Remove this")
		var needStock = updateLocationFlags.Bool("stockcheck", false, "Needs a stock check.")
		var inPlay = updateLocationFlags.Bool("inplay", false, "Is in play")
		var notInPlay = updateLocationFlags.Bool("notinplay", false, "Is in play")
		var physical = updateLocationFlags.Bool("physical", false, "Has physical media")
		var order = updateLocationFlags.Int("order", -1, "Has physical media")
		var gap = updateLocationFlags.Int("gap", -1, "Adds gaps")
		var adjust = updateLocationFlags.Bool("adjust", false, "Do adjust")
		var absWidth = updateLocationFlags.Float64("abs_width", -1, "Overall width")
		var absSlots = updateLocationFlags.Int("abs_slots", -1, "Slots")

		if err := updateLocationFlags.Parse(os.Args[2:]); err == nil {
			if *absSlots > 0 {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{
					Location: *name,
					Update:   &pb.Location{Quota: &pb.Quota{QuotaType: &pb.Quota_Slots{int32(*absSlots)}}},
				})

			}
			if *absWidth > 0 {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{
					Location: *name,
					Update:   &pb.Location{Quota: &pb.Quota{QuotaType: &pb.Quota_AbsoluteWidth{float32(*absWidth)}}},
				})
			}
			if *needStock {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{Checking: pb.Location_REQUIRE_STOCK_CHECK}})
			}
			if *adjust {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{AllowAdjust: true}})
			}
			if *delete {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, DeleteLocation: true})
			}
			if *gap > 0 {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{
					HardGap: map[int32]bool{int32(*gap): true},
				}})
			}
			if *folder > 0 && len(*sort) > 0 && *order >= 0 {
				if *sort == "label" {
					client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{
						FolderOrder: map[int32]int32{int32(*folder): int32(*order)},
						FolderSort:  map[int32]pb.Location_Sorting{int32(*folder): pb.Location_BY_LABEL_CATNO},
						FolderIds:   []int32{int32(*folder)}}})
				}
				if *sort == "release" {
					client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{
						FolderOrder: map[int32]int32{int32(*folder): int32(*order)},
						FolderSort:  map[int32]pb.Location_Sorting{int32(*folder): pb.Location_BY_RELEASE_DATE},
						FolderIds:   []int32{int32(*folder)}}})
				}
				if *sort == "time" {
					client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{
						FolderOrder: map[int32]int32{int32(*folder): int32(*order)},
						FolderSort:  map[int32]pb.Location_Sorting{int32(*folder): pb.Location_BY_DATE_ADDED},
						FolderIds:   []int32{int32(*folder)}}})
				}
				if *sort == "listen" {
					ur, err := client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{
						FolderOrder: map[int32]int32{int32(*folder): int32(*order)},
						FolderSort:  map[int32]pb.Location_Sorting{int32(*folder): pb.Location_BY_LAST_LISTEN},
						FolderIds:   []int32{int32(*folder)}}})
					fmt.Printf("%v, %v\n", ur, err)
				}
			}
			if *quota != 0 {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{Quota: &pb.Quota{NumOfSlots: int32(*quota)}}})
			}
			if *maxWidth != 0 {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{Quota: &pb.Quota{TotalWidth: float32(*maxWidth)}}})
			}

			if len(*sort) > 0 {
				if *sort == "time" {
					client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{Sort: pb.Location_BY_DATE_ADDED}})
				}
				if *sort == "folder" {
					client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{Sort: pb.Location_BY_FOLDER_THEN_DATE}})
				}

				if *sort == "move" {
					client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{Sort: pb.Location_BY_MOVE_TIME}})
				}
			}
			if !*alert {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{NoAlert: true}})
			}
			if *spill != 0 {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{SpillFolder: int32(*spill)}})
			}
			if *slots != 0 {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{Slots: int32(*slots)}})
			}
			if *optOut {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{OptOutQuotaChecks: true}})
			}
			if *reorgTime != 0 {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{ReorgTime: int64(*reorgTime)}})

			}
			if *inPlay {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{InPlay: pb.Location_IN_PLAY}})
			}
			if *notInPlay {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{InPlay: pb.Location_NOT_IN_PLAY}})
			}
			if *physical {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{MediaType: pb.Location_PHYSICAL}})
			}

		}
	case "extractor":
		extractFlags := flag.NewFlagSet("Extract", flag.ExitOnError)
		var label = extractFlags.Int("id", -1, "The ID of the label")
		var reg = extractFlags.String("extract", "", "The extractor")
		if err := extractFlags.Parse(os.Args[2:]); err == nil {
			client.AddExtractor(ctx, &pb.AddExtractorRequest{Extractor: &pb.LabelExtractor{LabelId: int32(*label), Extractor: *reg}})
		}
	}
}
