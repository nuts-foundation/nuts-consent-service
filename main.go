package main

import "github.com/nuts-foundation/nuts-consent-service/cmd"

func main() {

	println("nuts consent service")


	//// And now run an basic consent request:
	//id := uuid.New()
	//
	//// make sure the custodian has a keypair in the truststore
	//crypto := pkg.NewCryptoClient()
	//custodianID := "urn:oid:2.16.840.1.113883.2.4.6.1:123"
	//actorID := "urn:oid:2.16.840.1.113883.2.4.6.1:456"
	//keyID := types.KeyForEntity(types.LegalEntity{custodianID})
	//crypto.GenerateKeyPair(keyID)
	//
	//os.Setenv("NUTS_IDENTITY", "oid:123")
	//core.NutsConfig().Load(&cobra.Command{})
	//registryPath := "./registry"
	//r := pkg2.RegistryInstance()
	//r.Config.Mode = "server"
	//r.Config.Datadir = registryPath
	//r.Config.SyncMode = "fs"
	//r.Config.OrganisationCertificateValidity = 1
	//r.Config.VendorCACertificateValidity = 1
	//if err := r.Configure(); err != nil {
	//	panic(err)
	//}
	//
	//// Register a vendor
	//_, _ = r.RegisterVendor("Test Vendor", "healthcare")
	//
	//// Add Organization to registry
	//orgName := "Zorggroep Nuts"
	//if _, err := r.VendorClaim(actorID, orgName, nil); err != nil {
	//	//panic(err)
	//}
	//
	//proposeConsentCmd := &consentCommands.RegisterConsent{
	//	ID:          id,
	//	CustodianID: custodianID,
	//	SubjectID:   "bsn:999",
	//	ActorID:     "agb:456",
	//	Class:       "transfer",
	//	Start:       time.Now(),
	//}
	//
	//err = consentCommandHandler.HandleCommand(context.Background(), proposeConsentCmd)
	//if err != nil {
	//	log.Printf("[main] unable to handle command: %s\n", err)
	//}
	//
	////proposeConsentCmd.ID = uuid.New()
	////err = commandBus.HandleCommand(context.Background(), proposeConsentCmd)
	//
	//go func() {
	//	for e := range eventbus.Errors() {
	//		log.Printf("[eventbus] %s\n", e.Error())
	//	}
	//}()
	//
	//time.Sleep(5 * time.Second)
	//
	cmd.Execute()
	println("end")
}
