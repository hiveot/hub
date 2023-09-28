package corecli

// SetupCommand ensures the operating environment is correct.
// This:
// 1. if wipe option is given then delete all data and config
// 2. create missing folders
// 3. create the Hub self-signed CA if it doesn't exist
//
//	hubcli setup [--wipe] [--system]
//func SetupCommand(certsFolder *string) *cli.Command {
//	var force = false
//	var validityDays = 365 * 5
//
//	return &cli.Command{
//		Name:     "setup",
//		Usage:    "Setup the hiveot application environment",
//		Category: "core",
//		Flags: []cli.Flag{
//			&cli.IntFlag{
//				Name:        "days",
//				Usage:       "Number of `days` the certificate is valid.",
//				Value:       validityDays,
//				Destination: &validityDays,
//			},
//			&cli.BoolFlag{
//				Name:        "force",
//				Usage:       "Force overwrites an existing certificate and key.",
//				Destination: &force,
//			},
//		},
//		Action: func(cCtx *cli.Context) error {
//			if cCtx.NArg() > 0 {
//				return fmt.Errorf("unexpected argument(s) '%s'", cCtx.Args().First())
//			}
//			err := HandleSetup(*certsFolder, cCtx.Int("days"), cCtx.Bool("force"))
//			return err
//		},
//	}
//}
