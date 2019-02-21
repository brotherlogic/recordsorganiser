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

	pbgd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"

	//Needed to pull in gzip encoding init
	_ "google.golang.org/grpc/encoding/gzip"
)

// ByReleaseDate sorts by the given release date
type ByReleaseDate []*pbrc.Record

func (a ByReleaseDate) Len() int      { return len(a) }
func (a ByReleaseDate) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByReleaseDate) Less(i, j int) bool {
	if a[i].GetRelease().Released != a[j].GetRelease().Released {
		return a[i].GetRelease().Released > a[j].GetRelease().Released
	}
	return strings.Compare(a[i].GetRelease().Title, a[j].GetRelease().Title) < 0
}

func locateRelease(ctx context.Context, c pb.OrganiserServiceClient, id int32) {
	host, port, err := utils.Resolve("recordcollection")
	if err != nil {
		log.Fatalf("Unable to reach collection: %v", err)
	}
	conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		log.Fatalf("Unable to dial: %v", err)
	}

	client := pbrc.NewRecordCollectionServiceClient(conn)
	recs, err := client.GetRecords(ctx, &pbrc.GetRecordsRequest{Filter: &pbrc.Record{Release: &pbgd.Release{Id: id}}})
	if err != nil {
		log.Fatalf("Unable to get record %v -> %v", id, err)
	}

	if len(recs.GetRecords()) == 0 {
		fmt.Printf("No records with that id\n")
	}

	for _, rec := range recs.GetRecords() {
		location, err := c.Locate(ctx, &pb.LocateRequest{InstanceId: rec.GetRelease().InstanceId})
		if err != nil {
			fmt.Printf("Unable to locate instance (%v) of %v because %v\n", rec.GetRelease().InstanceId, rec.GetRelease().Title, err)
		} else {
			fmt.Printf("%v (%v) is in %v\n", rec.GetRelease().Title, rec.GetRelease().InstanceId, location.GetFoundLocation().GetName())

			for i, r := range location.GetFoundLocation().GetReleasesLocation() {
				if r.GetInstanceId() == rec.GetRelease().InstanceId {
					fmt.Printf("Slot %v\n", r.GetSlot())
					if i > 0 {
						fmt.Printf("%v. %v\n", i-1, getReleaseString(location.GetFoundLocation().GetReleasesLocation()[i-1]))
					}
					fmt.Printf("%v. %v\n", i, getReleaseString(location.GetFoundLocation().GetReleasesLocation()[i]))
					fmt.Printf("%v. %v\n", i+1, getReleaseString(location.GetFoundLocation().GetReleasesLocation()[i+1]))
				}
			}
		}
	}
}

func getReleaseString(loc *pb.ReleasePlacement) string {
	return loc.Title + " [" + strconv.Itoa(int(loc.InstanceId)) + "]"
}

func isTwelve(ctx context.Context, instanceID int32) bool {
	host, port, err := utils.Resolve("recordcollection")
	if err != nil {
		log.Fatalf("Unable to reach collection: %v", err)
	}
	conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		log.Fatalf("Unable to dial: %v", err)
	}

	client := pbrc.NewRecordCollectionServiceClient(conn)
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	rel, err := client.GetRecords(ctx, &pbrc.GetRecordsRequest{Filter: &pbrc.Record{Release: &pbgd.Release{InstanceId: instanceID}}})
	if err != nil {
		log.Fatalf("unable to get record (%v): %v", instanceID, err)
	}
	for _, f := range rel.GetRecords()[0].GetRelease().Formats {
		if f.Name == "LP" || f.Name == "Vinyl" {
			return true
		}
	}

	return false
}

func get(ctx context.Context, client pb.OrganiserServiceClient, name string, force bool, slot int32, twelves bool, reset bool) {
	locs, err := client.GetOrganisation(ctx, &pb.GetOrganisationRequest{OrgReset: reset, ForceReorg: force, Locations: []*pb.Location{&pb.Location{Name: name}}})
	if err != nil {
		log.Fatalf("Error reading locations: %v", err)
	}

	lastSlot := int32(1)
	for _, loc := range locs.GetLocations() {
		fmt.Printf("%v (%v) -> %v [%v] with %v (%v) %v [Last reorg: %v from %v]\n", loc.GetName(), len(loc.GetReleasesLocation()), loc.GetFolderIds(), loc.GetQuota(), loc.Sort.String(), loc.GetNoAlert(), loc.GetSpillFolder(), time.Unix(loc.LastReorg, 0), loc.ReorgTime)
		for j, rloc := range loc.GetReleasesLocation() {
			if slot < 0 || rloc.GetSlot() == slot {
				if !twelves || isTwelve(ctx, rloc.GetInstanceId()) {
					if rloc.GetSlot() > lastSlot {
						fmt.Printf("\n")
						lastSlot = rloc.GetSlot()
					}
					fmt.Printf("%v [%v]. %v\n", j, rloc.GetSlot(), getReleaseString(rloc))
				}
			}
		}
	}

	fmt.Printf("Summary: %v\n", locs.GetNumberProcessed())

	if len(locs.GetLocations()) == 0 {
		fmt.Printf("No Locations Found!\n")
	}
}

func list(ctx context.Context, client pb.OrganiserServiceClient) {
	locs, err := client.GetOrganisation(ctx, &pb.GetOrganisationRequest{Locations: []*pb.Location{&pb.Location{}}})
	if err != nil {
		log.Fatalf("Error reading locations: %v", err)
	}

	for i, loc := range locs.GetLocations() {
		fmt.Printf("%v. %v\n", i, loc.GetName())
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
	host, port, err := utils.Resolve("recordsorganiser")
	if err != nil {
		log.Fatalf("Unable to reach organiser: %v", err)
	}
	conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		log.Fatalf("Unable to dial: %v", err)
	}

	client := pb.NewOrganiserServiceClient(conn)
	ctx, cancel := utils.BuildContext("OrgCLI-"+os.Args[1], "recordsorganiser")
	defer cancel()

	switch os.Args[1] {
	case "list":
		list(ctx, client)
	case "get":
		getLocationFlags := flag.NewFlagSet("GetLocation", flag.ExitOnError)
		var name = getLocationFlags.String("name", "", "The name of the location")
		var force = getLocationFlags.Bool("force", false, "Force a reorg")
		var slot = getLocationFlags.Int("slot", 1, "Slot to view")
		var twelves = getLocationFlags.Bool("twelves", false, "Just 12 inches")
		var reorg = getLocationFlags.Bool("reorg", false, "Do a full reorg")

		if err := getLocationFlags.Parse(os.Args[2:]); err == nil {
			get(ctx, client, *name, *force, int32(*slot), *twelves, *reorg)
		}
	case "add":
		addLocationFlags := flag.NewFlagSet("AddLocation", flag.ExitOnError)
		var name = addLocationFlags.String("name", "", "The name of the new location")
		var slots = addLocationFlags.Int("slots", 0, "The number of slots in the location")
		var folderIds = addLocationFlags.String("folders", "", "The list of folder IDs")

		if err := addLocationFlags.Parse(os.Args[2:]); err == nil {
			nums := make([]int32, 0)
			for _, folderID := range strings.Split(*folderIds, ",") {
				v, err := strconv.Atoi(folderID)
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
					host, port, err := utils.Resolve("recordcollection")
					if err != nil {
						log.Fatalf("Unable to reach collection: %v", err)
					}
					conn, err := grpc.Dial(host+":"+strconv.Itoa(int(port)), grpc.WithInsecure())
					defer conn.Close()

					if err != nil {
						log.Fatalf("Unable to dial: %v", err)
					}

					client := pbrc.NewRecordCollectionServiceClient(conn)
					recs, err := client.GetRecords(ctx, &pbrc.GetRecordsRequest{Filter: &pbrc.Record{Release: &pbgd.Release{InstanceId: id}}})
					if err != nil {
						log.Fatalf("Unable to get record %v -> %v", id, err)
					}

					if len(recs.GetRecords()) == 0 {
						log.Fatalf("No records found", id)
					}
					fmt.Printf("%v\n", recs.GetRecords()[0].GetMetadata().Category)
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
			host, port, err := utils.Resolve("recordcollection")
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

				recs, err := rclient.GetRecords(ctx, &pbrc.GetRecordsRequest{Filter: &pbrc.Record{Release: &pbgd.Release{InstanceId: i}}})
				if err != nil {
					log.Fatalf("Error : %v", err)
				}

				for _, r := range recs.GetRecords() {
					records = append(records, r)
					score := float64(r.GetRelease().Rating)
					if score == 0 {
						score = float64(r.GetMetadata().OverallScore)
					}
					if score > 0.0 && score < minScore {
						minScore = score
					}
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
		var sort = updateLocationFlags.String("sort", "", "The new sorting mechanism")
		var alert = updateLocationFlags.Bool("alert", true, "Whether we should alert on this location")
		var spill = updateLocationFlags.Int("spill", 0, "The spill folder for this location")
		var slots = updateLocationFlags.Int("slots", 0, "The new number of slots for this location")
		var optOut = updateLocationFlags.Bool("out_out", false, "To opt this location out of quota alerts.")
		var reorgTime = updateLocationFlags.Int("reorg", 0, "The time needed to do a full reorg.")
		var delete = updateLocationFlags.Bool("delete", false, "Remove this")

		if err := updateLocationFlags.Parse(os.Args[2:]); err == nil {
			if *delete {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, DeleteLocation: true})
			}
			if *folder > 0 {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{FolderIds: []int32{int32(*folder)}}})
			}
			if *quota != 0 {
				client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{Quota: &pb.Quota{NumOfSlots: int32(*quota)}}})
			}
			if len(*sort) > 0 {
				if *sort == "time" {
					client.UpdateLocation(ctx, &pb.UpdateLocationRequest{Location: *name, Update: &pb.Location{Sort: pb.Location_BY_DATE_ADDED}})
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
