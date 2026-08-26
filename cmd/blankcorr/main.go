// Command blankcorr is the entry point for the geochemistry age-determination
// blank-correction service. In normal mode it serves the HTTP API; with
// --smoke-test it runs an end-to-end scenario, closes and reopens the
// database to verify persistence, and exits with a status code (0 = pass).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"task248-blankcorr/internal/httpapi"
	"task248-blankcorr/internal/model"
	"task248-blankcorr/internal/service"
	"task248-blankcorr/internal/store"
)

func main() {
	var (
		addr  string
		db    string
		smoke bool
	)
	flag.StringVar(&addr, "addr", ":8080", "HTTP listen address")
	flag.StringVar(&db, "db", "blankcorr.db", "SQLite database file path")
	flag.BoolVar(&smoke, "smoke-test", false, "run end-to-end smoke test and exit")
	flag.Parse()

	if smoke {
		if err := runSmokeTest(); err != nil {
			fmt.Fprintln(os.Stderr, "SMOKE TEST FAILED:", err)
			os.Exit(1)
		}
		fmt.Println("SMOKE TEST PASSED")
		os.Exit(0)
	}

	if err := runServer(addr, db); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func runServer(addr, dbPath string) error {
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	svc := service.New(st, 24*time.Hour)
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.NewServer(svc).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("blankcorr listening on %s (db=%s)", addr, dbPath)
	return srv.ListenAndServe()
}

// runSmokeTest exercises the full loop:
//   import batch + measurements -> match (picks a contaminated blank) ->
//   compute (anomaly) -> exclude bad blank -> recompute (clean age) ->
//   publish + seal -> close DB -> reopen -> verify persistence.
func runSmokeTest() error {
	f, err := os.CreateTemp("", "blankcorr-smoke-*.db")
	if err != nil {
		return fmt.Errorf("temp db: %w", err)
	}
	dbPath := f.Name()
	_ = f.Close()
	defer os.Remove(dbPath)

	ctx := context.Background()

	// --- first session: build, corrupt, then fix ---
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	svc := service.New(st, 24*time.Hour)

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(ms int64) time.Time { return base.Add(time.Duration(ms) * time.Millisecond) }

	b, err := svc.CreateBatch(ctx, "smoke-batch", "generic", 1e-4, 1.0, 900, 1300)
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	batchID := b.ID

	// standard: certified == measured -> recovery 1.0
	if _, _, err := svc.ImportMeasurement(ctx, model.MeasurementInput{
		BatchID: batchID, Kind: model.KindStandard, Material: "SRM-1",
		MeasuredAt: at(800), Ratio: 1.0, RatioUnc: 0.01, CertifiedRatio: 1.0,
	}); err != nil {
		return fmt.Errorf("import standard: %w", err)
	}
	// contaminated blank, nearest to the sample (t=1005 vs sample t=1000)
	cb, _, err := svc.ImportMeasurement(ctx, model.MeasurementInput{
		BatchID: batchID, Kind: model.KindBlank, Material: "reagent-A",
		MeasuredAt: at(1005), Ratio: 0.5, RatioUnc: 0.01,
	})
	if err != nil {
		return fmt.Errorf("import contaminated blank: %w", err)
	}
	// clean blank, further away (t=1200)
	if _, _, err := svc.ImportMeasurement(ctx, model.MeasurementInput{
		BatchID: batchID, Kind: model.KindBlank, Material: "reagent-A",
		MeasuredAt: at(1200), Ratio: 0.01, RatioUnc: 0.002,
	}); err != nil {
		return fmt.Errorf("import clean blank: %w", err)
	}
	// sample
	if _, _, err := svc.ImportMeasurement(ctx, model.MeasurementInput{
		BatchID: batchID, Kind: model.KindSample, Material: "unknown-X",
		MeasuredAt: at(1000), Ratio: 0.91, RatioUnc: 0.01,
	}); err != nil {
		return fmt.Errorf("import sample: %w", err)
	}

	// match -> initially binds the contaminated blank
	if _, err := svc.Match(ctx, batchID); err != nil {
		return fmt.Errorf("match: %w", err)
	}
	// compute -> should be anomalous (over-subtraction by bad blank)
	results, err := svc.ComputeAges(ctx, batchID)
	if err != nil {
		return fmt.Errorf("compute ages: %w", err)
	}
	if len(results) != 1 {
		return fmt.Errorf("expected 1 age result, got %d", len(results))
	}
	if !results[0].AnomalyFlag {
		return fmt.Errorf("expected anomaly from contaminated blank, got clean age %.1f", results[0].AgeValue)
	}
	log.Printf("smoke: detected anomaly age=%.1f (expected, due to bad blank)", results[0].AgeValue)

	// exclude the contaminated blank and recompute (re-match then correct)
	if _, err := svc.ExcludeMeasurement(ctx, cb.ID, "blank ratio 0.5 far above typical; contaminated", true); err != nil {
		return fmt.Errorf("exclude blank: %w", err)
	}
	results, err = svc.Recompute(ctx, batchID)
	if err != nil {
		return fmt.Errorf("recompute ages: %w", err)
	}
	if results[0].AnomalyFlag {
		return fmt.Errorf("after excluding bad blank, age still anomalous: %.1f (%s)", results[0].AgeValue, results[0].Reason)
	}
	log.Printf("smoke: recovered reasonable age=%.1f +/- %.1f (yr)", results[0].AgeValue, results[0].AgeUnc)

	// confirm the relation, publish and seal
	rels, err := svc.ListCorrections(ctx, batchID)
	if err != nil {
		return fmt.Errorf("list relations: %w", err)
	}
	if len(rels) != 1 {
		return fmt.Errorf("expected 1 relation, got %d", len(rels))
	}
	if _, err := svc.SetCorrectionStatus(ctx, rels[0].ID, model.CorrConfirmed); err != nil {
		return fmt.Errorf("confirm relation: %w", err)
	}
	ver, err := svc.PublishVersion(ctx, batchID, "v1", "initial published version", []int64{results[0].ID})
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	if _, err := svc.SealVersion(ctx, ver.ID); err != nil {
		return fmt.Errorf("seal: %w", err)
	}

	// --- verify persistence: close then reopen ---
	if err := st.Close(); err != nil {
		return fmt.Errorf("close store: %w", err)
	}
	st2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen store: %w", err)
	}
	defer st2.Close()
	svc2 := service.New(st2, 24*time.Hour)

	rb, err := svc2.GetBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("reopen get batch: %w", err)
	}
	if !rb.IsSealed() {
		return fmt.Errorf("batch not sealed after reopen")
	}
	ages, err := svc2.ListAgeResults(ctx, batchID)
	if err != nil {
		return fmt.Errorf("reopen list ages: %w", err)
	}
	if len(ages) != 1 {
		return fmt.Errorf("reopen: expected 1 age result, got %d", len(ages))
	}
	if ages[0].AnomalyFlag {
		return fmt.Errorf("reopen: age result flagged anomalous")
	}
	// the contaminated blank must remain excluded across restart
	blanks, err := svc2.ListMeasurements(ctx, batchID, model.KindBlank, []string{model.MeasContaminated})
	if err != nil {
		return fmt.Errorf("reopen list blanks: %w", err)
	}
	if len(blanks) != 1 {
		return fmt.Errorf("reopen: expected 1 contaminated blank, got %d", len(blanks))
	}
	sc, err := svc2.SelfCheck(ctx, batchID)
	if err != nil {
		return fmt.Errorf("self check: %w", err)
	}
	if len(sc.Problems) != 0 {
		return fmt.Errorf("self check reported problems after seal: %v", sc.Problems)
	}
	log.Printf("smoke: persistence verified (batch sealed, %d age result, %d contaminated blank)", len(ages), len(blanks))
	return nil
}
