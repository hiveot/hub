package main

const testSource1 = "../../runtime/digitwin/tm/Values.json"
const testSource2 = "../../runtime/digitwin/tm/Directory.json"
const testSource3 = "../../runtime/authn/tm/admin.json"
const testSource4 = "../../runtime/authn/tm/user.json"

// Helper for testing the types generator
// func Test_types(t *testing.T) {
// 	logging.SetLogging("info", "")
// 	cwd, _ := os.Getwd()
// 	sourceFiles := []string{path.Join(cwd, testSource1),
// 		path.Join(cwd, testSource2),
// 		path.Join(cwd, testSource3),
// 		path.Join(cwd, testSource4),
// 	}

// 	outDir := path.Join(os.TempDir(), "test-types")
// 	force := true
// 	err := GenerateSources("types", sourceFiles, outDir, force)
// 	assert.NoError(t, err)
// 	main()
// }
