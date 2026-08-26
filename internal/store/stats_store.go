package store

import "fmt"

// Stats holds aggregate counts across the whole database, used by the
// /api/stats endpoint.
type Stats struct {
	Batches     int `json:"batches"`
	Measurements int `json:"measurements"`
	Samples     int `json:"samples"`
	Blanks      int `json:"blanks"`
	Standards   int `json:"standards"`
	Corrections int `json:"corrections"`
	AgeResults  int `json:"age_results"`
	Anomalies   int `json:"anomalies"`
	Versions    int `json:"versions"`
	Exclusions  int `json:"exclusions"`
}

// CountStats returns aggregate counts over the database.
func (s *Store) CountStats() (*Stats, error) {
	count := func(q string) (int, error) {
		var n int
		if err := s.db.QueryRow(q).Scan(&n); err != nil {
			return 0, fmt.Errorf("count %q: %w", q, err)
		}
		return n, nil
	}
	st := &Stats{}
	var err error
	if st.Batches, err = count(`SELECT COUNT(*) FROM batches`); err != nil {
		return nil, err
	}
	if st.Measurements, err = count(`SELECT COUNT(*) FROM measurements`); err != nil {
		return nil, err
	}
	if st.Samples, err = count(`SELECT COUNT(*) FROM measurements WHERE kind='sample'`); err != nil {
		return nil, err
	}
	if st.Blanks, err = count(`SELECT COUNT(*) FROM measurements WHERE kind='blank'`); err != nil {
		return nil, err
	}
	if st.Standards, err = count(`SELECT COUNT(*) FROM measurements WHERE kind='standard'`); err != nil {
		return nil, err
	}
	if st.Corrections, err = count(`SELECT COUNT(*) FROM correction_relations`); err != nil {
		return nil, err
	}
	if st.AgeResults, err = count(`SELECT COUNT(*) FROM age_results`); err != nil {
		return nil, err
	}
	if st.Anomalies, err = count(`SELECT COUNT(*) FROM age_results WHERE anomaly_flag=1`); err != nil {
		return nil, err
	}
	if st.Versions, err = count(`SELECT COUNT(*) FROM age_versions`); err != nil {
		return nil, err
	}
	if st.Exclusions, err = count(`SELECT COUNT(*) FROM exclusions`); err != nil {
		return nil, err
	}
	return st, nil
}
