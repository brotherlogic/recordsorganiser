package sales

import (
	"io/ioutil"
	"log"
	"sort"
	"testing"

	pbgd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	"github.com/golang/protobuf/proto"

	"github.com/brotherlogic/goserver/utils"
)

var orderData = []struct {
	in  []*pbrc.Record
	out []*pbrc.Record
}{
	{
		// Lower priced records before higher
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{CurrentSalePrice: 100}},
			&pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{CurrentSalePrice: 50}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{CurrentSalePrice: 50}},
			&pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{CurrentSalePrice: 100}},
		},
	},
	{
		// Later priced record before higher
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{CurrentSalePrice: 50}},
			&pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{CurrentSalePrice: 100}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{CurrentSalePrice: 50}},
			&pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{CurrentSalePrice: 100}},
		},
	},

	{
		// Scores before rating
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{Rating: 5, Released: "2002"}, Metadata: &pbrc.ReleaseMetadata{}},
			&pbrc.Record{Release: &pbgd.Release{Released: "2001"}, Metadata: &pbrc.ReleaseMetadata{OverallScore: 4}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{Released: "2001"}, Metadata: &pbrc.ReleaseMetadata{OverallScore: 4}},
			&pbrc.Record{Release: &pbgd.Release{Released: "2002", Rating: 5}, Metadata: &pbrc.ReleaseMetadata{}},
		},
	},
	{
		// Lower scores before higher
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{Rating: 5}, Metadata: &pbrc.ReleaseMetadata{}},
			&pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{OverallScore: 4}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{OverallScore: 4}},
			&pbrc.Record{Release: &pbgd.Release{Rating: 5}, Metadata: &pbrc.ReleaseMetadata{}},
		},
	},
	{
		// Lower scores before higher
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{OverallScore: 4}},
			&pbrc.Record{Release: &pbgd.Release{Rating: 5}, Metadata: &pbrc.ReleaseMetadata{}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{}, Metadata: &pbrc.ReleaseMetadata{OverallScore: 4}},
			&pbrc.Record{Release: &pbgd.Release{Rating: 5}, Metadata: &pbrc.ReleaseMetadata{}},
		},
	},
	{
		// Later releases should be placed before earlier ones
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{Released: "2001"}, Metadata: &pbrc.ReleaseMetadata{}},
			&pbrc.Record{Release: &pbgd.Release{Released: "2002"}, Metadata: &pbrc.ReleaseMetadata{}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{Released: "2002"}, Metadata: &pbrc.ReleaseMetadata{}},
			&pbrc.Record{Release: &pbgd.Release{Released: "2001"}, Metadata: &pbrc.ReleaseMetadata{}},
		},
	},
	{
		// Later releases should be placed before earlier ones
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{Released: "2002"}, Metadata: &pbrc.ReleaseMetadata{}},
			&pbrc.Record{Release: &pbgd.Release{Released: "2001"}, Metadata: &pbrc.ReleaseMetadata{}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Release: &pbgd.Release{Released: "2002"}, Metadata: &pbrc.ReleaseMetadata{}},
			&pbrc.Record{Release: &pbgd.Release{Released: "2001"}, Metadata: &pbrc.ReleaseMetadata{}},
		},
	},
	{
		// MATCH_UNKNOWN shoule come after FULL_MATCH
		[]*pbrc.Record{
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Match: pbrc.ReleaseMetadata_FULL_MATCH}, Release: &pbgd.Release{}},
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Match: pbrc.ReleaseMetadata_MATCH_UNKNOWN}, Release: &pbgd.Release{}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Match: pbrc.ReleaseMetadata_FULL_MATCH}, Release: &pbgd.Release{}},
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Match: pbrc.ReleaseMetadata_MATCH_UNKNOWN}, Release: &pbgd.Release{}},
		},
	},
	{
		// MATCH_UNKNOWN shoule come after FULL_MATCH
		[]*pbrc.Record{
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Match: pbrc.ReleaseMetadata_MATCH_UNKNOWN}, Release: &pbgd.Release{}},
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Match: pbrc.ReleaseMetadata_FULL_MATCH}, Release: &pbgd.Release{}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Match: pbrc.ReleaseMetadata_FULL_MATCH}, Release: &pbgd.Release{}},
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Match: pbrc.ReleaseMetadata_MATCH_UNKNOWN}, Release: &pbgd.Release{}},
		},
	},
	{
		// NOT_KEEPER should come before KEEPER
		[]*pbrc.Record{
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_KEEPER}},
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_NOT_KEEPER}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_NOT_KEEPER}},
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_KEEPER}},
		},
	},
	{
		// NOT_KEEPER should come before KEEPER
		[]*pbrc.Record{
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_NOT_KEEPER}},
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_KEEPER}},
		},
		[]*pbrc.Record{
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_NOT_KEEPER}},
			&pbrc.Record{Metadata: &pbrc.ReleaseMetadata{Keep: pbrc.ReleaseMetadata_KEEPER}},
		},
	}}

func TestSaleOrdering(t *testing.T) {
	for _, entry := range orderData {
		sort.Sort(BySaleOrder(entry.in))
		for i := range entry.in {
			if utils.FuzzyMatch(entry.in[i], entry.out[i]) != nil {
				t.Errorf("Sorting error: %v vs %v", entry.in[i], entry.out[i])
			}
		}
	}
}

func TestSaleSpecificOrdering(t *testing.T) {
	data, _ := ioutil.ReadFile("testdata/11427909.data")
	data2, _ := ioutil.ReadFile("testdata/13180489.data")

	record1 := &pbrc.Record{}
	record2 := &pbrc.Record{}
	err := proto.Unmarshal(data, record1)
	if err != nil {
		t.Fatalf("Cannot read data :%v", err)
	}
	log.Printf("Found %v", record1)

	err = proto.Unmarshal(data2, record2)
	if err != nil {
		t.Fatalf("Cannot read data :%v", err)
	}

	records := []*pbrc.Record{record1, record2}
	log.Printf("%v", records[0].GetRelease().GetId())
	sort.Sort(BySaleOrder(records))
	log.Printf("%v", records[0].GetRelease().GetId())

	if records[0].GetRelease().GetId() == 11427909 {
		t.Errorf("Bad ordering")
	}
}

func TestSaleSpecificOrderingpart2(t *testing.T) {
	data, _ := ioutil.ReadFile("testdata/3077034.data")
	data2, _ := ioutil.ReadFile("testdata/13180489.data")

	record1 := &pbrc.Record{}
	record2 := &pbrc.Record{}
	err := proto.Unmarshal(data, record1)
	if err != nil {
		t.Fatalf("Cannot read data :%v", err)
	}
	err = proto.Unmarshal(data2, record2)
	if err != nil {
		t.Fatalf("Cannot read data :%v", err)
	}

	records := []*pbrc.Record{record1, record2}
	log.Printf("BEF %v", record1)
	sort.Sort(BySaleOrder(records))
	log.Printf("AFT %v", records[0].GetRelease().GetId())

	if records[0].GetRelease().GetId() != 13180489 {
		t.Errorf("Bad ordering")
	}
}
