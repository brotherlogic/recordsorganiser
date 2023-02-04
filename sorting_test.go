package main

import (
	"fmt"
	"sort"
	"testing"

	pbd "github.com/brotherlogic/godiscogs/proto"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pb "github.com/brotherlogic/recordsorganiser/proto"
	"golang.org/x/net/context"
)

func testLog(ctx context.Context, s string) {
	fmt.Printf("%v\n", s)
}

func TestSortByDateAdded(t *testing.T) {
	releases := []*pbrc.Record{
		&pbrc.Record{Release: &pbd.Release{Id: 2}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 125}},
		&pbrc.Record{Release: &pbd.Release{Id: 3}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 124}},
		&pbrc.Record{Release: &pbd.Release{Id: 4}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 123}},
	}

	sort.Sort(ByDateAdded(releases))

	if releases[0].Release.Id != 4 {
		t.Errorf("Releases are not correctly ordered: %v", releases)
	}
}

func TestSortByDateAddedProper(t *testing.T) {
	releases := []*pbrc.Record{
		&pbrc.Record{Release: &pbd.Release{Id: 2}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 1516927655}},
		&pbrc.Record{Release: &pbd.Release{Id: 3}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 1521687761}},
		&pbrc.Record{Release: &pbd.Release{Id: 2}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 1}},
		&pbrc.Record{Release: &pbd.Release{Id: 4}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 1520384637}},
	}

	sort.Sort(ByDateAdded(releases))

	for i := 1; i < len(releases); i++ {
		if releases[i].Metadata.DateAdded < releases[i-1].Metadata.DateAdded {
			t.Fatalf("Error in sorting by date added: %v", releases)
		}
	}
}

func TestSortByLabelCat(t *testing.T) {
	releases := []*pbrc.Record{
		&pbrc.Record{Release: &pbd.Release{Id: 2, Labels: []*pbd.Label{&pbd.Label{Name: "TestOne"}}}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 125}},
		&pbrc.Record{Release: &pbd.Release{Id: 3, Labels: []*pbd.Label{&pbd.Label{Name: "TestTwo"}}}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 124}},
		&pbrc.Record{Release: &pbd.Release{Id: 4, Labels: []*pbd.Label{&pbd.Label{Name: "TestA"}}}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 123}},
	}

	sort.Sort(ByLabelCat{releases, make(map[int32]string), testLog, &pb.SortingCache{}})

	if releases[0].Release.Id != 4 {
		t.Errorf("Releases are not correctly ordered: %v", releases)
	}
}

func TestSortByLabelCatWhenCatImbalance(t *testing.T) {
	releases := []*pbrc.Record{
		&pbrc.Record{Release: &pbd.Release{Id: 2, Labels: []*pbd.Label{&pbd.Label{Name: "TestOne", Catno: "1234-"}}}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 125}},
		&pbrc.Record{Release: &pbd.Release{Id: 3, Labels: []*pbd.Label{&pbd.Label{Name: "TestOne", Catno: "1234-1234"}}}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 124}},
		&pbrc.Record{Release: &pbd.Release{Id: 4, Labels: []*pbd.Label{&pbd.Label{Name: "TestA"}}}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 123}},
	}

	sort.Sort(ByLabelCat{releases, make(map[int32]string), testLog, &pb.SortingCache{}})

	if releases[0].Release.Id != 4 {
		t.Errorf("Releases are not correctly ordered: %v", releases)
	}
}

func TestSortByLabelCatNoLabels(t *testing.T) {
	releases := []*pbrc.Record{
		&pbrc.Record{Release: &pbd.Release{Id: 2}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 125}},
		&pbrc.Record{Release: &pbd.Release{Id: 3, Labels: []*pbd.Label{&pbd.Label{Name: "TestOne", Catno: "1234-1234"}}}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 124}},
		&pbrc.Record{Release: &pbd.Release{Id: 4}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 123}},
	}

	sort.Sort(ByLabelCat{releases, make(map[int32]string), testLog, &pb.SortingCache{}})

	if releases[0].Release.Id != 4 {
		t.Errorf("Releases are not correctly ordered: %v", releases)
	}
}

func TestSortByDateAddedWithFallback(t *testing.T) {
	releases := []*pbrc.Record{
		&pbrc.Record{Release: &pbd.Release{Title: "Second", Id: 2}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 124}},
		&pbrc.Record{Release: &pbd.Release{Title: "Third", Id: 3}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 124}},
		&pbrc.Record{Release: &pbd.Release{Title: "First", Id: 4}, Metadata: &pbrc.ReleaseMetadata{DateAdded: 124}},
	}

	sort.Sort(ByDateAdded(releases))

	if releases[0].Release.Id != 4 {
		t.Errorf("Releases are not correctly ordered: %v", releases)
	}
}

var sortTests = []struct {
	r1 pbd.Release
	r2 pbd.Release
}{
	{pbd.Release{Labels: []*pbd.Label{&pbd.Label{Name: "TestOne"}}},
		pbd.Release{Labels: []*pbd.Label{&pbd.Label{Name: "TestTwo"}}}},

	{pbd.Release{Title: "Low", Labels: []*pbd.Label{&pbd.Label{Name: "TestOne"}}},
		pbd.Release{Title: "VeryLow", Labels: []*pbd.Label{&pbd.Label{Name: "TestOne"}}}},

	{pbd.Release{Labels: []*pbd.Label{&pbd.Label{Name: "TestOne", Catno: "First"}}},
		pbd.Release{Labels: []*pbd.Label{&pbd.Label{Name: "TestOne", Catno: "Second"}}}},

	{pbd.Release{Labels: []*pbd.Label{&pbd.Label{Name: "TestOne", Catno: "IM 2"}}},
		pbd.Release{Labels: []*pbd.Label{&pbd.Label{Name: "TestOne", Catno: "IM 12"}}}},
}

var defaultComp = []struct {
	r1 pbd.Release
	r2 pbd.Release
}{
	{pbd.Release{Labels: []*pbd.Label{&pbd.Label{Name: "TestOne"}}},
		pbd.Release{Labels: []*pbd.Label{&pbd.Label{Name: "TestOne"}}}},
}

func TestSortingByLabelCat(t *testing.T) {
	for _, tt := range sortTests {
		sValue := sortByLabelCat(&tt.r1, &tt.r2, make(map[int32]string), testLog, &pb.SortingCache{})
		if sValue >= 0 {
			t.Errorf("%v should come before %v (%v)", tt.r1, tt.r2, sValue)
		}
		sValueR := sortByLabelCat(&tt.r2, &tt.r1, make(map[int32]string), testLog, &pb.SortingCache{})
		if sValueR <= 0 {
			t.Errorf("%v should come before %v (%v)", tt.r1, tt.r2, sValueR)
		}
	}

	tt := defaultComp[0]
	sValue := sortByLabelCat(&tt.r1, &tt.r2, make(map[int32]string), testLog, &pb.SortingCache{})
	sValue2 := sortByLabelCat(&tt.r2, &tt.r1, make(map[int32]string), testLog, &pb.SortingCache{})
	if sValue != 0 || sValue2 != 0 {
		t.Errorf("Default is not zero: %v and %v", sValue, sValue2)
	}
}

func TestSortByMasterReleaseDate(t *testing.T) {
	releases := []*pbrc.Record{
		&pbrc.Record{Release: &pbd.Release{Id: 2, EarliestReleaseDate: 15}},
		&pbrc.Record{Release: &pbd.Release{Id: 3, EarliestReleaseDate: 10}},
		&pbrc.Record{Release: &pbd.Release{Id: 4, EarliestReleaseDate: 20}},
		&pbrc.Record{Release: &pbd.Release{Id: 5, EarliestReleaseDate: 15}},
	}

	sort.Sort(ByEarliestReleaseDate(releases))

	if releases[0].GetRelease().Id != 3 {
		t.Errorf("Releases are not correctly ordered: %v", releases)
	}
}

func TestSortByFolderThenMasterReleaseDate(t *testing.T) {
	releases := []*pbrc.Record{
		&pbrc.Record{Release: &pbd.Release{Id: 2, FolderId: 1, EarliestReleaseDate: 15}},
		&pbrc.Record{Release: &pbd.Release{Id: 3, FolderId: 1, EarliestReleaseDate: 10}},
		&pbrc.Record{Release: &pbd.Release{Id: 4, FolderId: 2, EarliestReleaseDate: 20}},
		&pbrc.Record{Release: &pbd.Release{Id: 5, FolderId: 2, Title: "yay", EarliestReleaseDate: 15}},
		&pbrc.Record{Release: &pbd.Release{Id: 6, FolderId: 2, Title: "nay", EarliestReleaseDate: 15}},
	}

	sort.Sort(ByFolderThenRelease(releases))

	if releases[0].GetRelease().Id != 3 ||
		releases[1].GetRelease().Id != 2 ||
		releases[2].GetRelease().Id != 6 ||
		releases[3].GetRelease().Id != 5 {
		t.Errorf("Releases are not correctly ordered: %v", releases)
	}
}

func TestGetFormatWidth(t *testing.T) {
	v := getFormatWidth(&pbrc.Record{Release: &pbd.Release{FormatQuantity: 1, Labels: []*pbd.Label{&pbd.Label{Name: "Death Waltz Recording Company"}}}}, 2.0)
	if v != 2.0 {
		t.Errorf("Bad width: %v", v)
	}
}

func TestGetFormatWidthForNowAgain(t *testing.T) {
	v := getFormatWidth(&pbrc.Record{Release: &pbd.Release{FormatQuantity: 1, Labels: []*pbd.Label{&pbd.Label{Name: "Now-Again Records"}}}}, 2.0)
	if v != 2.0 {
		t.Errorf("Bad width: %v", v)
	}
}

func TestGetFormatWidthForBox(t *testing.T) {
	v := getFormatWidth(&pbrc.Record{Release: &pbd.Release{FormatQuantity: 4, Formats: []*pbd.Format{&pbd.Format{Text: "Boxset"}}}}, 2.0)
	if v != 2.0 {
		t.Errorf("Bad width: %v", v)
	}
}

func TestGetFormatWidthForGatefold(t *testing.T) {
	v := getFormatWidth(&pbrc.Record{Release: &pbd.Release{FormatQuantity: 1, Formats: []*pbd.Format{&pbd.Format{Text: "Gatefold"}}}}, 2.0)
	if v != 2.0 {
		t.Errorf("Bad width: %v", v)
	}
}

func TestExtractorSplit(t *testing.T) {
	m := make(map[int32]string)
	m[int32(123)] = "(5\\d\\d)"
	vals := doExtractorSplit(&pbd.Label{Catno: "MPI-503", Id: 123}, m, testLog)

	if len(vals) != 1 {
		t.Errorf("Bad extraction: %v", vals)
	}
}

func TestExtractorSplitBadExtractor(t *testing.T) {
	m := make(map[int32]string)
	m[int32(123)] = "(5\\d\\d"
	vals := doExtractorSplit(&pbd.Label{Catno: "MPI-503", Id: 123}, m, testLog)

	if len(vals) != 0 {
		t.Errorf("Bad extraction: %v", vals)
	}
}

func TestExtractorSplitNoCandidates(t *testing.T) {
	m := make(map[int32]string)
	vals := doExtractorSplit(&pbd.Label{Catno: "MPI-503", Id: 123}, m, testLog)

	if len(vals) != 0 {
		t.Errorf("Bad extraction: %v", vals)
	}
}

func TestLoadedSorting(t *testing.T) {
	tests := []struct {
		r1 int32
		r2 int32
	}{
		{r1: 494740378, r2: 492447790},
	}

	for _, te := range tests {
		for sw := 0; sw <= 1; sw++ {
			r1 := loadTestRecord(te.r1)
			r2 := loadTestRecord(te.r2)

			if r1 == nil || r2 == nil {
				t.Fatalf("Unable to load records")
			}

			cache := &pb.SortingCache{}
			entry1 := appendCache(cache, r1)
			entry2 := appendCache(cache, r2)
			if sw == 1 {
				entry1, entry2 = entry2, entry1
			}

			val := sortByLabelCatCached(entry1, entry2, cache)

			if (sw == 0 && val != -1) || (sw == 1 && val != 1) {
				t.Errorf("Test is poorly ordered: %v, %v => %v", entry1, entry2, val)
			}
		}
	}
}
