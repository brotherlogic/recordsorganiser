package main

import (
	"fmt"
	"sort"
	"testing"

	pbd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
)

func testLog(s string) {
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

	sort.Sort(ByLabelCat{releases, make(map[int32]string), testLog})

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

	sort.Sort(ByLabelCat{releases, make(map[int32]string), testLog})

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

	sort.Sort(ByLabelCat{releases, make(map[int32]string), testLog})

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
		sValue := sortByLabelCat(&tt.r1, &tt.r2, make(map[int32]string), testLog)
		if sValue >= 0 {
			t.Errorf("%v should come before %v (%v)", tt.r1, tt.r2, sValue)
		}
		sValueR := sortByLabelCat(&tt.r2, &tt.r1, make(map[int32]string), testLog)
		if sValueR <= 0 {
			t.Errorf("%v should come before %v (%v)", tt.r1, tt.r2, sValueR)
		}
	}

	tt := defaultComp[0]
	sValue := sortByLabelCat(&tt.r1, &tt.r2, make(map[int32]string), testLog)
	sValue2 := sortByLabelCat(&tt.r2, &tt.r1, make(map[int32]string), testLog)
	if sValue != 0 || sValue2 != 0 {
		t.Errorf("Default is not zero: %v and %v", sValue, sValue2)
	}
}

func TestSortByMasterReleaseDate(t *testing.T) {
	releases := []*pbd.Release{
		&pbd.Release{Id: 2, EarliestReleaseDate: 15},
		&pbd.Release{Id: 3, EarliestReleaseDate: 10},
		&pbd.Release{Id: 4, EarliestReleaseDate: 20},
		&pbd.Release{Id: 5, EarliestReleaseDate: 15},
	}

	sort.Sort(ByEarliestReleaseDate(releases))

	if releases[0].Id != 3 {
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
	v := getFormatWidth(&pbrc.Record{Release: &pbd.Release{FormatQuantity: 1, Labels: []*pbd.Label{&pbd.Label{Name: "Death Waltz Recording Company"}}}})
	if v != 2.0 {
		t.Errorf("Bad width: %v", v)
	}
}

func TestGetFormatWidthForNowAgain(t *testing.T) {
	v := getFormatWidth(&pbrc.Record{Release: &pbd.Release{FormatQuantity: 1, Labels: []*pbd.Label{&pbd.Label{Name: "Now-Again Records"}}}})
	if v != 2.0 {
		t.Errorf("Bad width: %v", v)
	}
}

func TestGetFormatWidthForBox(t *testing.T) {
	v := getFormatWidth(&pbrc.Record{Release: &pbd.Release{FormatQuantity: 4, Formats: []*pbd.Format{&pbd.Format{Text: "Boxset"}}}})
	if v != 5.0 {
		t.Errorf("Bad width: %v", v)
	}
}

func TestGetFormatWidthForGatefold(t *testing.T) {
	v := getFormatWidth(&pbrc.Record{Release: &pbd.Release{FormatQuantity: 1, Formats: []*pbd.Format{&pbd.Format{Text: "Gatefold"}}}})
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

func TestBasicSplit(t *testing.T) {
	releases := []*pbrc.Record{&pbrc.Record{Release: &pbd.Release{FormatQuantity: 1}}, &pbrc.Record{Release: &pbd.Release{FormatQuantity: 1}}}

	s := getTestServer(".testbasicsplit")
	splits := s.Split(releases, 2)

	if len(splits) != 2 {
		t.Errorf("Bad split: %v", len(splits))
	}
}
