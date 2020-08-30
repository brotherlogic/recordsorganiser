package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	pb "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
)

// ByDateAdded allows sorting of releases by the date they were added
type ByDateAdded []*pbrc.Record

func (a ByDateAdded) Len() int      { return len(a) }
func (a ByDateAdded) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByDateAdded) Less(i, j int) bool {
	if a[i].Metadata.DateAdded != a[j].Metadata.DateAdded {
		return a[i].Metadata.DateAdded < a[j].Metadata.DateAdded
	}
	return strings.Compare(a[i].Release.Title, a[j].Release.Title) < 0
}

// ByLabelCat allows sorting of releases by the date they were added
type ByLabelCat struct {
	records    []*pbrc.Record
	extractors map[int32]string
	logger     func(string)
}

func (a ByLabelCat) Len() int      { return len(a.records) }
func (a ByLabelCat) Swap(i, j int) { a.records[i], a.records[j] = a.records[j], a.records[i] }
func (a ByLabelCat) Less(i, j int) bool {
	return sortByLabelCat(a.records[i].GetRelease(), a.records[j].GetRelease(), a.extractors, a.logger) < 0
}

func split(str string) []string {
	return regexp.MustCompile("[0-9]+|[a-z]+|[A-Z]+").FindAllString(str, -1)
}

func doExtractorSplit(label *pb.Label, ex map[int32]string, logger func(string)) []string {
	if val, ok := ex[label.Id]; ok {
		r, err := regexp.Compile(val)
		if err != nil {
			return make([]string, 0)
		}
		vals := r.FindAllStringSubmatch(label.Catno, -1)
		ret := make([]string, 0)
		for _, pair := range vals {
			ret = append(ret, pair[1])
		}
		logger(fmt.Sprintf("RAN ON %v got %v", label, ret))
		return ret
	}

	return make([]string, 0)
}

// Sorts by label and then catalogue number
func sortByLabelCat(rel1, rel2 *pb.Release, extractors map[int32]string, logger func(string)) int {
	if len(rel1.Labels) == 0 {
		return -1
	}
	if len(rel2.Labels) == 0 {
		return 1
	}

	label1 := pb.GetMainLabel(rel1.Labels)
	label2 := pb.GetMainLabel(rel2.Labels)

	labelSort := strings.Compare(label1.Name, label2.Name)
	if labelSort != 0 {
		return labelSort
	}

	cat1Elems := doExtractorSplit(label1, extractors, logger)
	cat2Elems := doExtractorSplit(label2, extractors, logger)

	if len(cat1Elems) == 0 || len(cat1Elems) != len(cat2Elems) {
		cat1Elems = split(label1.Catno)
		cat2Elems = split(label2.Catno)
	}

	toCheck := len(cat1Elems)
	if len(cat2Elems) < toCheck {
		toCheck = len(cat2Elems)
	}

	for i := 0; i < toCheck; i++ {
		if unicode.IsNumber(rune(cat1Elems[i][0])) && unicode.IsNumber(rune(cat2Elems[i][0])) {
			num1, _ := strconv.Atoi(cat1Elems[i])
			num2, _ := strconv.Atoi(cat2Elems[i])
			if num1 > num2 {
				return 1
			} else if num2 > num1 {
				return -1
			}
		} else {
			catComp := strings.Compare(cat1Elems[i], cat2Elems[i])
			if catComp != 0 {
				return catComp
			}
		}
	}

	//Fallout to sorting by title
	titleComp := strings.Compare(rel1.Title, rel2.Title)
	return titleComp
}

// ByEarliestReleaseDate allows sorting by the earliest release date
type ByEarliestReleaseDate []*pb.Release

func (a ByEarliestReleaseDate) Len() int      { return len(a) }
func (a ByEarliestReleaseDate) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByEarliestReleaseDate) Less(i, j int) bool {
	if a[i].EarliestReleaseDate != a[j].EarliestReleaseDate {
		return a[i].EarliestReleaseDate < a[j].EarliestReleaseDate
	}
	return strings.Compare(a[i].Title, a[j].Title) < 0
}

// ByFolderThenRelease allows sorting by the earliest release date
type ByFolderThenRelease []*pbrc.Record

func (a ByFolderThenRelease) Len() int      { return len(a) }
func (a ByFolderThenRelease) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByFolderThenRelease) Less(i, j int) bool {
	if a[i].GetRelease().FolderId != a[j].GetRelease().FolderId {
		return a[i].GetRelease().FolderId < a[j].GetRelease().FolderId
	}

	if a[i].GetRelease().EarliestReleaseDate != a[j].GetRelease().EarliestReleaseDate {
		return a[i].GetRelease().EarliestReleaseDate < a[j].GetRelease().EarliestReleaseDate
	}
	return strings.Compare(a[i].GetRelease().Title, a[j].GetRelease().Title) < 0
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

// Split splits a releases list into buckets
func (s *Server) Split(releases []*pbrc.Record, n float32) [][]*pbrc.Record {
	var solution [][]*pbrc.Record

	var count float32
	count = 0
	for _, rel := range releases {
		count += getFormatWidth(rel)
	}

	boundaryAccumulator := count / n
	boundaryValue := boundaryAccumulator
	currentValue := float32(0.0)
	var currentReleases []*pbrc.Record
	for _, rel := range releases {
		if currentValue+getFormatWidth(rel) > boundaryValue {
			solution = append(solution, currentReleases)
			currentReleases = make([]*pbrc.Record, 0)
			boundaryValue += boundaryAccumulator
		}

		currentReleases = append(currentReleases, rel)
		currentValue += getFormatWidth(rel)
	}
	solution = append(solution, currentReleases)

	return solution
}
