package sales

import (
	"math"
	"strings"

	pbrc "github.com/brotherlogic/recordcollection/proto"
)

//BySaleOrder - the order in which we sell things
type BySaleOrder []*pbrc.Record

// Get The Score
func GetScore(r *pbrc.Record) float32 {
	// Treat NaN as a zero score
	if math.IsNaN(float64(r.GetMetadata().OverallScore)) {
		return 0
	}

	return r.GetMetadata().OverallScore
}

func (a BySaleOrder) Len() int      { return len(a) }
func (a BySaleOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a BySaleOrder) Less(i, j int) bool {
	// Sort by score
	if a[i].GetMetadata() != nil && a[j].GetMetadata() != nil {
		if GetScore(a[i]) != GetScore(a[j]) {
			return GetScore(a[i]) < GetScore(a[j])
		}
	}

	// Sort by current price
	if a[i].GetMetadata() != nil && a[j].GetMetadata() != nil {
		if a[i].GetMetadata().CurrentSalePrice != a[j].GetMetadata().CurrentSalePrice {
			return a[i].GetMetadata().CurrentSalePrice < a[j].GetMetadata().CurrentSalePrice
		}
	}

	if a[i].GetRelease().Released != a[j].GetRelease().Released {

		return a[i].GetRelease().Released > a[j].GetRelease().Released
	}

	return strings.Compare(a[i].GetRelease().Title, a[j].GetRelease().Title) < 0
}
