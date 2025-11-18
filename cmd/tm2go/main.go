package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hiveot/gocore/logging"
	"github.com/hiveot/gocore/wot/td"
	"github.com/hiveot/hub/cmd/tm2go/genagent"
	"github.com/hiveot/hub/cmd/tm2go/genconsumer"
	"github.com/hiveot/hub/cmd/tm2go/gentypes"
	"github.com/hiveot/hub/cmd/tm2go/listtms"
	"github.com/urfave/cli/v2"
)

const TypesAPISuffix = "Types.go"
const AgentAPISuffix = "AgentAPI.go"
const ConsumerAPISuffix = "ConsumerAPI.go"

const Version = `0.3-alpha`
const TMDir = "tm"
const APIDir = "api"

// CLI for generating API's from Thing Model/Description Documents (TM)
func main() {
	logging.SetLogging("warning", "")
	var recursive bool
	//var outFile = ""
	var outDir = path.Join("..", APIDir)
	var force bool

	app := &cli.App{
		EnableBashCompletion: true,
		Name:                 "tm2go",
		Usage:                "HiveOT API code generator for golang from a TD document definitions",
		Version:              Version,

		// commands arguments are passed by reference so they are updated in the Before section
		Commands: []*cli.Command{

			&cli.Command{
				Name:      "list",
				Usage:     "List the TD's available in all packages",
				UsageText: "list [package-directory]",
				Action: func(cCtx *cli.Context) error {
					packageDir, _ := filepath.Abs(cCtx.Args().First())
					return listtms.HandleTMScan(packageDir)
				},
			},
			&cli.Command{
				Name:  "generate",
				Usage: "Generate golang code from a TD.",
				Args:  true,
				UsageText: "generate [-f] [-r] types|agent|consumer|all source" +
					"\nWhere:" +
					"\n  types    instructs to generate only the constants and dataschema types from the TD." +
					"\n  agent    instructs to generate the service api and request handler code." +
					"\n  consumer instructs to generate the client api code." +
					"\n  all      instructs to generate types, agent and consumer code." +
					"\n  source   is the path to the TD document or 'tdd' directory holding one or more sources.",
				UseShortOptionHandling: true,

				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:        "recursive",
						Aliases:     []string{"r"},
						Usage:       "recurse to find 'tdd' subdirectories",
						Value:       recursive,
						Destination: &recursive,
					},
					&cli.BoolFlag{
						Name:        "force",
						Usage:       "force generate even when up to date",
						Aliases:     []string{"f"},
						Value:       force,
						Destination: &force,
					},
					&cli.StringFlag{
						Name:        "outdir",
						Usage:       "output directory to generate sources, relative to sourcefile",
						Value:       outDir,
						Destination: &outDir,
					},
				},
				Action: func(cCtx *cli.Context) (err error) {
					if cCtx.NArg() < 2 {
						return fmt.Errorf("Expected a source type and a source file or directory")
					}
					genType := cCtx.Args().First()
					// Determine the package directory (used as agentID)
					sourceFiles := []string{}
					source, _ := filepath.Abs(cCtx.Args().Get(1))
					sourceStat, err := os.Stat(source)
					if err != nil {
						return err
					}
					if sourceStat.IsDir() {
						// source is a directory with source files
						sourceFiles = LocateSources(source, recursive)
					} else {
						// source is the file
						sourceFiles = append(sourceFiles, source)
					}
					if genType == "all" {
						err = GenerateSources("types", sourceFiles, outDir, force)
						if err == nil {
							err = GenerateSources("agent", sourceFiles, outDir, force)
						}
						if err == nil {
							err = GenerateSources("consumer", sourceFiles, outDir, force)
						}
					} else {
						err = GenerateSources(genType, sourceFiles, outDir, force)
					}
					if err != nil {
						slog.Error("generate error", "err", err.Error())
					}
					return err
				},
			},
		},
	}
	app.Suggest = true
	app.HideHelpCommand = true
	if err := app.Run(os.Args); err != nil {
		println("ERROR: ", err.Error())
		os.Exit(1)
		//helpArgs := append(os.Args, "-h")
		//_ = app.Run(helpArgs)
	}
}

// LocateSources locates all tm/file.json sources in the given directory or below
func LocateSources(rootDir string, recursive bool) (sources []string) {
	sources = make([]string, 0)

	if recursive {
		// recursively iterate all directories looking for tm/*.json
		_ = filepath.Walk(rootDir, func(path string, finfo os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("Error: %s\n", err.Error())
				return err
			}
			// look for .json files in tdd directories
			if !finfo.IsDir() {
				sourceDir := filepath.Dir(path)
				if strings.HasSuffix(sourceDir, TMDir) {
					if strings.HasSuffix(finfo.Name(), "json") {
						sources = append(sources, path)
					}
				}
			}
			return nil
		})
	} else {
		// locate all the .json files in the given rootDir
		entries, err := os.ReadDir(rootDir)
		if err != nil {
			return sources
		}
		for _, entry := range entries {
			finfo, _ := entry.Info()
			if finfo.IsDir() {
				// looking for files, not subdirectories
				continue
			}
			name := finfo.Name()
			if strings.HasSuffix(path.Ext(name), TMDir) {
				fullPath := filepath.Join(rootDir, name)
				sources = append(sources, fullPath)
			}
		}
	}
	return sources
}

// GenerateSources finds TDs and generate the source code in its api directory
func GenerateSources(gentype string, tdFiles []string, outDir string, force bool) (err error) {

	for _, tdFile := range tdFiles {
		sourceDir := filepath.Dir(tdFile)
		packageDir := filepath.Dir(sourceDir)
		agentID := filepath.Base(packageDir)
		outDirAbs := outDir
		if !filepath.IsAbs(outDir) {
			outDirAbs = filepath.Join(sourceDir, outDir)
		}
		err = GenerateSource(gentype, agentID, tdFile, outDirAbs, force)
		if err != nil {
			return err
		}
	}
	return err
}

// GenerateSource generates source code for the given TD JSON file
//
//	gentype is one of {types, agent, consumer} to generate - also used as filename suffix.
//	agentID is the service agent ID
//	tdi is the TD to save
//	fileName is the name of the json input and used for the go output file
//	outDir is the directory to write the file in
//	force generate file even when existing output is newer
//
// The package name is the parent directory name, for example: 'digitwin' in digitwin/tm/directory.json
func GenerateSource(gentype string, agentID string, sourceFile string, outDir string, force bool) error {
	var err error
	var outfilePath string

	tdi, err := td.ReadTDFromFile(sourceFile)
	if err != nil {
		err = fmt.Errorf("GenerateSource failed: %w", err)
		return err
	}
	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		return err
	}

	// get the output name without file extension
	outFileName := filepath.Base(sourceFile)
	outFileName = strings.TrimSuffix(outFileName, filepath.Ext(outFileName))
	outFileName = gentypes.ToTitle(outFileName)

	sourceStat, _ := os.Stat(sourceFile)
	l := &gentypes.SL{}
	if gentype == "types" {
		outfilePath = filepath.Join(outDir, outFileName+TypesAPISuffix)
		outfileStat, _ := os.Stat(outfilePath)
		if !force && outfileStat != nil && outfileStat.ModTime().After(sourceStat.ModTime()) {
			fmt.Printf("GenerateSource: Destination %s is already up to date. Not updated.\n", outfilePath)
			return nil
		} else {
			err = gentypes.GenTypes(l, agentID, tdi)
		}

	} else if gentype == "agent" {
		outfilePath = filepath.Join(outDir, outFileName+AgentAPISuffix)
		outfileStat, _ := os.Stat(outfilePath)
		if !force && outfileStat != nil && outfileStat.ModTime().After(sourceStat.ModTime()) {
			fmt.Printf("GenerateSource: Destination %s is newer. Not updated.\n", outfilePath)
			return nil
		} else {
			err = genagent.GenAgent(l, agentID, tdi)
		}

	} else if gentype == "consumer" {
		outfilePath = filepath.Join(outDir, outFileName+ConsumerAPISuffix)
		outfileStat, _ := os.Stat(outfilePath)
		if !force && outfileStat != nil && outfileStat.ModTime().After(sourceStat.ModTime()) {
			fmt.Printf("GenerateSource: Destination %s is newer. Not updated.\n", outfilePath)
			return nil
		} else {
			err = genconsumer.GenServiceConsumer(l, agentID, tdi)
		}

	} else {
		err = fmt.Errorf("GenerateSource: unknown generate command '%s'", gentype)
	}
	if err != nil {
		return err
	}
	// 4: Save the types
	err = l.Write(outfilePath)
	if err != nil {
		fmt.Printf("GenerateSource: Failed to generated %s into %s: %s\n", gentype, outfilePath, err.Error())
	} else {
		fmt.Printf("GenerateSource: Generated %s into %s\n", gentype, outfilePath)
	}
	return err
}
