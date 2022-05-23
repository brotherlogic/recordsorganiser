package locator

import (
	"context"
	"fmt"
	"strconv"

	"github.com/brotherlogic/goserver/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbrc "github.com/brotherlogic/recordcollection/proto"
	pbro "github.com/brotherlogic/recordsorganiser/proto"
)

func getRecord(ctx context.Context, client pbrc.RecordCollectionServiceClient, id int32) (*pbrc.Record, error) {
	val, err := client.GetRecord(ctx, &pbrc.GetRecordRequest{InstanceId: id})
	if err != nil {
		return nil, err
	}
	return val.GetRecord(), err
}

func getReleaseString(ctx context.Context, client pbrc.RecordCollectionServiceClient, loc *pbro.ReleasePlacement, showSleeve bool) string {
	rec, err := getRecord(ctx, client, loc.InstanceId)
	if err != nil {
		return fmt.Sprintf("%v", err)
	}
	/*if rec.GetMetadata().GetFiledUnder() == pbrc.ReleaseMetadata_FILE_DIGITAL || rec.GetMetadata().GetFiledUnder() == pbrc.ReleaseMetadata_FILE_CD {
		return ""
	}*/
	sleeve := ""
	if showSleeve {
		sleeve = fmt.Sprintf("%v", rec.GetMetadata().GetSleeve())
	}
	return fmt.Sprintf("%v. ", rec.GetRelease().GetId()) + loc.Title + " " + fmt.Sprintf("%v", rec.GetMetadata().GetFiledUnder()) + " [" + strconv.Itoa(int(loc.InstanceId)) + "] - " + fmt.Sprintf("%v", rec.GetMetadata().GetCategory()) + " {" + fmt.Sprintf("%v", loc.GetDeterminedWidth()) + "} + " + fmt.Sprintf("%v", rec.GetMetadata().GetLastMoveTime()) + " [" + fmt.Sprintf("%v", rec.GetRelease().GetLabels()) + "]" + sleeve
}

func ReadableLocation(ctx context.Context, id int32) (string, error) {
	conn, err := utils.LFDialServer(ctx, "recordcollection")

	if err != nil {
		return "", err
	}
	defer conn.Close()
	client := pbrc.NewRecordCollectionServiceClient(conn)

	conn2, err := utils.LFDialServer(ctx, "recordsorganiser")
	if err != nil {
		return "", err
	}
	c := pbro.NewOrganiserServiceClient(conn2)

	location, err := c.Locate(ctx, &pbro.LocateRequest{InstanceId: id})
	if err != nil {
		return "", err
	}

	rec, err := client.GetRecord(ctx, &pbrc.GetRecordRequest{InstanceId: id})
	if err != nil {
		return "", err
	}
	fmt.Printf("%v (%v) is in %v\n", rec.GetRecord().GetRelease().Title, rec.GetRecord().GetRelease().InstanceId, location.GetFoundLocation().GetName())

	for i, r := range location.GetFoundLocation().GetReleasesLocation() {
		str := ""
		if r.GetInstanceId() == id {
			str += fmt.Sprintf("Slot %v\n", r.GetSlot())
			if i > 0 {
				str += fmt.Sprintf("%v. %v\n", i-1, getReleaseString(ctx, client, location.GetFoundLocation().GetReleasesLocation()[i-1], false))
			}
			str += fmt.Sprintf("%v. %v\n", i, getReleaseString(ctx, client, location.GetFoundLocation().GetReleasesLocation()[i], false))
			str += fmt.Sprintf("%v. %v\n", i+1, getReleaseString(ctx, client, location.GetFoundLocation().GetReleasesLocation()[i+1], false))
		}
	}

	return "", status.Errorf(codes.NotFound, "Could not locate %v", id)
}
