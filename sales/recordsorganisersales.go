package sales

import (
	"strings"

	pbrc "github.com/brotherlogic/recordcollection/proto"
)

//BySaleOrder - the order in which we sell things
type BySaleOrder []*pbrc.Record

func getScore(r *pbrc.Record) float32 {
	if r.GetRelease().Rating != 0 {
		return float32(r.GetRelease().Rating)
	}
	return r.GetMetadata().OverallScore
}

func (a BySaleOrder) Len() int      { return len(a) }
func (a BySaleOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a BySaleOrder) Less(i, j int) bool {
	if a[i].GetMetadata() != nil && a[j].GetMetadata() != nil {
		if a[i].GetMetadata().Keep != a[j].GetMetadata().Keep {
			if a[i].GetMetadata().Keep == pbrc.ReleaseMetadata_KEEPER {
				return false
			}
			if a[j].GetMetadata().Keep == pbrc.ReleaseMetadata_KEEPER {
				return true
			}
		}
	}

	// Push FULL_MATCH first
	if a[i].GetMetadata() != nil && a[j].GetMetadata() != nil {
		if a[i].GetMetadata().Match != a[j].GetMetadata().Match {
			if a[i].GetMetadata().Match == pbrc.ReleaseMetadata_FULL_MATCH {
				return true
			}
			if a[j].GetMetadata().Match == pbrc.ReleaseMetadata_FULL_MATCH {
				return false
			}
		}
	}

	// Sort by score
	if a[i].GetMetadata() != nil && a[j].GetMetadata() != nil {
		if getScore(a[i]) != getScore(a[j]) {
			return getScore(a[i]) < getScore(a[j])
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
