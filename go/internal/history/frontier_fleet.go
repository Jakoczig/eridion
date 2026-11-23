package history

import "context"

// ReconciledFleetCount is the Commander "known owned ships" count. It is the
// owned total of the same resolved inventory that drives Fleet rows, search
// and paging, so the two can never disagree. facts may be nil.
func (s *Store) ReconciledFleetCount(ctx context.Context, fid string, facts []EffectiveFact) (int, error) {
	fleet, e := s.ResolvedFleet(ctx, fid, facts)
	return fleet.Owned, e
}
