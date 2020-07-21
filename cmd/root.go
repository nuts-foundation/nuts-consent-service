package cmd

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nuts-foundation/nuts-consent-service/api"
	engine2 "github.com/nuts-foundation/nuts-consent-service/engine"
	pkg2 "github.com/nuts-foundation/nuts-consent-service/pkg"
	engine3 "github.com/nuts-foundation/nuts-consent-store/engine"
	engine4 "github.com/nuts-foundation/nuts-event-octopus/engine"
	core "github.com/nuts-foundation/nuts-go-core"
	"github.com/nuts-foundation/nuts-registry/engine"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

const confPort = "port"
const confInterface = "interface"
const version = `Nuts consent logic v0.1 -- HEAD`

var serveCommand = &cobra.Command{
	Use:   "serve",
	Short: "Start consent-service as a standalone api server",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Print: " + strings.Join(args, " "))
		server := echo.New()
		server.HideBanner = true
		server.Use(middleware.Logger())
		instance := pkg2.ConsentServiceInstance()
		api.RegisterHandlers(server, api.Wrapper{Cl: instance})
		addr := fmt.Sprintf("%s:%d", serverInterface, serverPort)
		server.Logger.Fatal(server.Start(addr))
	},
}
var (
	serverInterface string
	serverPort      int
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	nutsConfig := core.NutsConfig()

	var consentLogicEngine = engine2.NewConsentServiceEngine()

	var rootCommand = consentLogicEngine.Cmd
	serveCommand.Flags().StringVar(&serverInterface, confInterface, "localhost", "Server interface binding")
	serveCommand.Flags().IntVarP(&serverPort, confPort, "p", 1324, "Server listen port")
	rootCommand.AddCommand(serveCommand)

	nutsConfig.IgnoredPrefixes = append(nutsConfig.IgnoredPrefixes, consentLogicEngine.ConfigKey)
	nutsConfig.RegisterFlags(rootCommand, consentLogicEngine)

	registryEngine := engine.NewRegistryEngine()
	nutsConfig.RegisterFlags(rootCommand, registryEngine)

	consentStoreEngine := engine3.NewConsentStoreEngine()
	nutsConfig.RegisterFlags(rootCommand, consentStoreEngine)

	eventOctopusEngine := engine4.NewEventOctopusEngine()
	nutsConfig.RegisterFlags(rootCommand, eventOctopusEngine)

	if err := nutsConfig.Load(rootCommand); err != nil {
		panic(err)
	}

	nutsConfig.PrintConfig(logrus.StandardLogger())

	if err := nutsConfig.InjectIntoEngine(consentLogicEngine); err != nil {
		panic(err)
	}

	if err := nutsConfig.InjectIntoEngine(registryEngine); err != nil {
		panic(err)
	}

	if err := nutsConfig.InjectIntoEngine(consentStoreEngine); err != nil {
		panic(err)
	}

	if err := consentLogicEngine.Configure(); err != nil {
		panic(err)
	}

	if err := eventOctopusEngine.Configure(); err != nil {
		panic(err)
	}

	if err := eventOctopusEngine.Start(); err != nil {
		panic(err)
	}

	if err := registryEngine.Configure(); err != nil {
		panic(err)
	}

	if err := consentLogicEngine.Start(); err != nil {
		panic(err)
	}

	if err := rootCommand.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}