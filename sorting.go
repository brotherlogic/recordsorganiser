package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/brotherlogic/godiscogs"

	pb "github.com/brotherlogic/godiscogs/proto"
	pbrc "github.com/brotherlogic/recordcollection/proto"
	pbro "github.com/brotherlogic/recordsorganiser/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/net/context"

	"github.com/fvbommel/sortorder"
)

// ByDateAdded allows sorting of releases by the date they were added
type ByIID []*pbrc.Record

func (a ByIID) Len() int      { return len(a) }
func (a ByIID) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByIID) Less(i, j int) bool {
	return a[i].GetRelease().GetInstanceId() < a[j].GetRelease().GetInstanceId()
}

// ByDateAdded allows sorting of releases by the date they were added
type ByLastListen []*pbrc.Record

func (a ByLastListen) Len() int      { return len(a) }
func (a ByLastListen) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLastListen) Less(i, j int) bool {
	if a[i].Metadata.LastListenTime != a[j].Metadata.LastListenTime {
		return a[i].Metadata.LastListenTime < a[j].Metadata.LastListenTime
	}
	return strings.Compare(a[i].Release.Title, a[j].Release.Title) < 0
}

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

// ByDateMoved allows sorting of releases by the date they were added
type ByDateMoved []*pbrc.Record

func (a ByDateMoved) Len() int      { return len(a) }
func (a ByDateMoved) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByDateMoved) Less(i, j int) bool {
	if a[i].Metadata.LastMoveTime != a[j].Metadata.LastMoveTime {
		return a[i].Metadata.LastMoveTime < a[j].Metadata.LastMoveTime
	}
	return strings.Compare(a[i].Release.Title, a[j].Release.Title) < 0
}

// ByLabelCat allows sorting of releases by the date they were added
type ByLabelCat struct {
	records    []*pbrc.Record
	extractors map[int32]string
	logger     func(context.Context, string)
	cache      *pbro.SortingCache
}

func (a ByLabelCat) Len() int      { return len(a.records) }
func (a ByLabelCat) Swap(i, j int) { a.records[i], a.records[j] = a.records[j], a.records[i] }
func (a ByLabelCat) Less(i, j int) bool {
	return sortByLabelCat(a.records[i].GetRelease(), a.records[j].GetRelease(), a.extractors, a.logger, a.cache) < 0
}

// ByLabelCat allows sorting of releases by the date they were added
type ByCachedLabelCat struct {
	records []int32
	cache   *pbro.SortingCache
}

func (a ByCachedLabelCat) Len() int      { return len(a.records) }
func (a ByCachedLabelCat) Swap(i, j int) { a.records[i], a.records[j] = a.records[j], a.records[i] }
func (a ByCachedLabelCat) Less(i, j int) bool {
	return sortByLabelCatCached(getEntry(a.cache, a.records[i]), getEntry(a.cache, a.records[j]), a.cache) < 0
}

func split(str string) []string {
	return regexp.MustCompile("[0-9]+|[a-z]+|[A-Z]+").FindAllString(str, -1)
}

func doExtractorSplit(label *pb.Label, ex map[int32]string, logger func(context.Context, string)) []string {
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
		return ret
	}

	return make([]string, 0)
}

// Sorts by label and then catalogue number
func sortByLabelCatCached(rel1, rel2 *pbro.CacheEntry, cache *pbro.SortingCache) int {
	bits1 := strings.Split(rel1.GetEntry()["BY_LABEL"], "|")
	bits2 := strings.Split(rel2.GetEntry()["BY_LABEL"], "|")

	val := strings.Compare(bits1[0], bits2[0])
	if val != 0 {
		return val
	}

	if sortorder.NaturalLess(bits1[1], bits2[1]) {
		return -1
	}
	if bits1[1] == bits2[1] {
		return strings.Compare(bits1[2], bits2[2])
	}
	return 1
}

// Sorts by label and then catalogue number
func sortByLabelCat(rel1, rel2 *pb.Release, extractors map[int32]string, logger func(context.Context, string), cache *pbro.SortingCache) int {

	if len(rel1.Labels) == 0 {
		return -1
	}
	if len(rel2.Labels) == 0 {
		return 1
	}

	label1 := godiscogs.GetMainLabel(rel1.Labels)
	label2 := godiscogs.GetMainLabel(rel2.Labels)

	labelSort := strings.Compare(strings.ToLower(label1.GetName()), strings.ToLower(label2.GetName()))
	if labelSort != 0 {
		return labelSort
	}

	cat1Elems := doExtractorSplit(label1, extractors, logger)
	cat2Elems := doExtractorSplit(label2, extractors, logger)

	if len(cat1Elems) == 0 || len(cat1Elems) != len(cat2Elems) {
		cat1Elems = split(strings.ToLower(label1.Catno))
		cat2Elems = split(strings.ToLower(label2.Catno))
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
type ByEarliestReleaseDate []*pbrc.Record

func (a ByEarliestReleaseDate) Len() int      { return len(a) }
func (a ByEarliestReleaseDate) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByEarliestReleaseDate) Less(i, j int) bool {
	if a[i].GetRelease().EarliestReleaseDate != a[j].GetRelease().EarliestReleaseDate {
		return a[i].GetRelease().EarliestReleaseDate < a[j].GetRelease().EarliestReleaseDate
	}
	return strings.Compare(a[i].GetRelease().Title, a[j].GetRelease().Title) < 0
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

func getFormatWidth(r *pbrc.Record, bwidth float64) float32 {
	// Use the spine width if we have it
	if r.GetMetadata().GetRecordWidth() > 0 {
		// Make the adjustment for DS_F records
		if r.GetMetadata().GetSleeve() == pbrc.ReleaseMetadata_BAGS_UNLIMITED_PLAIN ||
			r.GetMetadata().GetSleeve() == pbrc.ReleaseMetadata_VINYL_STORAGE_DOUBLE_FLAP {
			return r.GetMetadata().GetRecordWidth() * 1.26
		}

		if r.GetMetadata().GetSleeve() == pbrc.ReleaseMetadata_SLEEVE_UNKNOWN {
			return r.GetMetadata().GetRecordWidth() * 1.18
		}

		if r.GetMetadata().GetSleeve() == pbrc.ReleaseMetadata_VINYL_STORAGE_NO_INNER {
			return r.GetMetadata().GetRecordWidth() * 1.4
		}

		return r.GetMetadata().GetRecordWidth()
	}

	return float32(bwidth)
}

var (
	fstart = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "recordsorganiser_slot_start",
		Help: "Various Wait Times",
	}, []string{"location", "folder"})
)

// Split splits a releases list into buckets
func (s *Server) Split(ctx context.Context, loc string, releases []*pbrc.Record, n float32, maxw float32, hardgap []int, allowAdjust bool, bwidth float64) [][]*pbrc.Record {
	var solution [][]*pbrc.Record

	var counts []float32
	count := float32(0)
	tslots := 0
	for i, rel := range releases {
		for _, gap := range hardgap {
			if i == gap {
				nslots := int(math.Ceil(float64(count) / float64(maxw)))
				tslots += nslots
				poss := float32(count/float32(nslots)) + 10.0
				if poss > maxw {
					poss = maxw
				}
				counts = append(counts, maxw)
				count = 0
			}
		}
		count += getFormatWidth(rel, bwidth)
	}

	counts = append(counts, maxw)

	s.CtxLog(ctx, fmt.Sprintf("HERE %v: %v", loc, counts))

	version := 0
	currentValue := float32(0.0)
	var currentReleases []*pbrc.Record
	for i := range releases {
		found := false
		for _, gap := range hardgap {
			if i == gap {
				found = true
			}
		}
		if found {
			s.CtxLog(ctx, fmt.Sprintf("Flipping Found hard gap for %v", loc))
			solution = append(solution, currentReleases)
			currentReleases = make([]*pbrc.Record, 0)
			currentValue = 0
			version++
		} else if currentValue+getFormatWidth(releases[i], bwidth) > counts[version] {

			s.CtxLog(ctx, fmt.Sprintf("Flipping %v @ %v / %v, because %v + %v is greater than %v -> %v", loc, len(solution), i, currentValue, getFormatWidth(releases[i], bwidth), counts[version], counts))

			if allowAdjust && i < len(releases)-1 && currentValue+getFormatWidth(releases[i+1], bwidth) < counts[version] {
				releases[i], releases[i+1] = releases[i+1], releases[i]
			} else if allowAdjust && i < len(releases)-2 && currentValue+getFormatWidth(releases[i+2], bwidth) < counts[version] {
				releases[i], releases[i+2] = releases[i+2], releases[i]
				releases[i+1], releases[i+2] = releases[i+2], releases[i+1] // Correct misorder
			} else if allowAdjust && i < len(releases)-3 && currentValue+getFormatWidth(releases[i+3], bwidth) < counts[version] {
				releases[i], releases[i+3] = releases[i+3], releases[i]
				releases[i+1], releases[i+3] = releases[i+3], releases[i+1] // Correct misorder
				releases[i+2], releases[i+3] = releases[i+3], releases[i+2] // Correct misorder
			} else {
				solution = append(solution, currentReleases)
				currentReleases = make([]*pbrc.Record, 0)
				currentValue = 0
			}
		}

		currentReleases = append(currentReleases, releases[i])
		currentValue += getFormatWidth(releases[i], bwidth)

	}
	solution = append(solution, currentReleases)

	return solution
}
