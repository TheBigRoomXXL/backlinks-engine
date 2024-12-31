package planer

// func init() {
// 	os.Setenv("DB_NAME", "test")
// 	postgres, err := database.New()
// 	if err != nil {
// 		log.Fatalf("failed to init tests: %s", err)
// 	}

// 	_, err = postgres.Pool.Exec(context.Background(), `
// 		DROP DATABASE IF EXISTS test;
// 		CREATE DATABASE test;
// 	`)
// 	if err != nil {
// 		log.Fatalf("failed to clear db for tests: %s", err)
// 	}
// }

// func TestSeed(t *testing.T) {
// 	planner, err := New()
// 	if err != nil {
// 		t.Fatalf("test crashed: %s", err)
// 	}
// 	planner.Seed([]string{"http://localhost/seeds"})

// 	seeds, err := planner.Next()
// 	if err != nil {
// 		t.Fatalf("test crashed: %s", err)
// 	}

// 	if len(seeds) != 1 {
// 		t.Fatalf("invalide number of seed: got %d, expect 1", len(seeds))
// 	}

// 	s := seeds[0]
// 	if s.Scheme != "http" && s.Host != "localhost" && s.Path != "/seeds" {
// 		t.Fatalf("invalid seed: got %s; expected http://localhost/seeds", s.String())
// 	}
// }
