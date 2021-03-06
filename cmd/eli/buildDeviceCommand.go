package main

import (
	"fmt"
	"io"
	"os"

	"github.com/ernoaapa/eliot/pkg/cmd"
	"github.com/ernoaapa/eliot/pkg/cmd/build"
	"github.com/ernoaapa/eliot/pkg/cmd/ui"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var buildDeviceCommand = cli.Command{
	Name:    "device",
	Aliases: []string{"dev", "devices"},
	Usage:   "Build device image",
	UsageText: `eli build device [options] [FILE | URL]
	
	 # Build default device image
	 eli build device
	 
	 # Create Linuxkit file but don't build it
	 eli build device --dry-run
	 eli build device --dry-run > custom-linuxkit.yml
	 
	 # Build from custom config and unpack to directory
	 mkdir dist
	 eli build device custom-linuxkit.yml | tar xv -C dist
	 `,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "dry-run",
			Usage: "Print the final Linuxkit config and don't actually build it",
		},
		cli.StringFlag{
			Name:   "build-server",
			Usage:  "Linuxkit build server (github.com/ernoaapa/linuxkit-server) base url",
			Value:  "http://build.eliot.run",
			EnvVar: "ELIOT_BUILD_SERVER",
		},
		cli.StringFlag{
			Name:  "output",
			Usage: "Target output file",
			Value: "image.tar",
		},
		cli.StringFlag{
			Name:  "type",
			Usage: "Target build type, one of Linuxkit output types",
			Value: "rpi3",
		},
	},
	Action: func(clicontext *cli.Context) (err error) {
		var (
			source     = clicontext.Args().First()
			dryRun     = clicontext.Bool("dry-run")
			serverURL  = clicontext.String("build-server")
			outputFile = clicontext.String("output")
			outputType = clicontext.String("type")

			uiline   ui.Line
			linuxkit []byte
			output   io.Writer
		)

		uiline = ui.NewLine().Loading("Get Linuxkit config...")
		linuxkit, err = build.ResolveLinuxkitConfig(source)
		if err != nil {
			uiline.Errorf("Failed to resolve Linuxkit config: %s", err)
			return errors.Wrap(err, "Cannot resolve Linuxkit config")
		}
		uiline.Done("Resolved Linuxkit config!")

		uiline = ui.NewLine().Loading("Resolve output...")
		if cmd.IsPipingOut() {
			output = os.Stdout
			uiline.Done("Resolved output to stdout!")
		} else {
			outFile, err := os.Create(outputFile)
			if err != nil {
				uiline.Errorf("Error, cannot create target output file %s", outputFile)
				return fmt.Errorf("Cannot create target output file %s", outputFile)
			}
			defer outFile.Close()
			output = outFile
			uiline.Donef("Resolved output: %s!", outFile.Name())
		}

		if dryRun {
			fmt.Println(string(linuxkit))
			return nil
		}

		uiline = ui.NewLine().Loadingf("Building Linuxkit image in remote build server...")
		image, err := build.BuildImage(serverURL, outputType, linuxkit)
		if err != nil {
			uiline.Errorf("Failed to build Linuxkit image: %s", err)
			return errors.Wrap(err, "Failed to build Linuxkit image")
		}

		uiline.Loadingf("Write Linuxkit image to output...")
		_, err = io.Copy(output, image)
		if err != nil {
			uiline.Errorf("Error while writing output: %s", err)
			return errors.New("Unable to copy image to output")
		}

		uiline.Donef("Build complete!")
		return nil
	},
}
