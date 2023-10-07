package provcli

import (
	"context"
	"fmt"
	"github.com/hiveot/hub/pkg/provisioning"
	"github.com/hiveot/hub/pkg/provisioning/capnpclient"

	"github.com/urfave/cli/v2"

	"github.com/hiveot/hub/lib/hubclient"
)

// ProvisionAddOOBSecretsCommand
// prov add  <deviceID> <oobsecret>
func ProvisionAddOOBSecretsCommand(ctx context.Context, runFolder *string) *cli.Command {
	return &cli.Command{
		Name:      "addsecret",
		Usage:     "Add a out of band provisioning secret for a Thing",
		ArgsUsage: "<deviceID> <secret>",
		Category:  "provisioning",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 2 {
				return fmt.Errorf("expected 2 arguments. Got %d instead", cCtx.NArg())
			}
			err := HandleAddOobSecret(ctx, *runFolder,
				cCtx.Args().Get(0),
				cCtx.Args().Get(1))
			fmt.Println("Adding secret for device: ", cCtx.Args().First())
			return err
		},
	}
}

// ProvisionApproveRequestCommand
// prov approve <deviceID>
func ProvisionApproveRequestCommand(ctx context.Context, runFolder *string) *cli.Command {
	return &cli.Command{
		Name:      "approve",
		Usage:     "Approve a pending provisioning request",
		ArgsUsage: "<deviceID>",
		Category:  "provisioning",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 1 {
				return fmt.Errorf("expected 1 arguments. Got %d instead", cCtx.NArg())
			}
			deviceID := cCtx.Args().First()
			err := HandleApproveRequest(ctx, *runFolder, deviceID)
			return err
		},
	}
}

// ProvisionGetApprovedRequestsCommand
// prov approved
func ProvisionGetApprovedRequestsCommand(ctx context.Context, runFolder *string) *cli.Command {
	return &cli.Command{
		Name:     "lapproved",
		Usage:    "List approved provisioning requests",
		Category: "provisioning",
		Action: func(cCtx *cli.Context) error {
			err := HandleGetApprovedRequests(ctx, *runFolder)
			return err
		},
	}
}

// ProvisionGetPendingRequestsCommand
// prov approved
func ProvisionGetPendingRequestsCommand(ctx context.Context, runFolder *string) *cli.Command {
	return &cli.Command{
		Name:     "lpending",
		Usage:    "List pending provisioning requests",
		Category: "provisioning",
		Action: func(cCtx *cli.Context) error {
			err := HandleGetPendingRequests(ctx, *runFolder)
			return err
		},
	}
}

// HandleAddOobSecret invokes the out-of-band provisioning service to add a provisioning secret
//
//	deviceID is the ID of the device whose secret to set
//	secret to set
func HandleAddOobSecret(ctx context.Context, runFolder string, deviceID string, secret string) error {
	var pc provisioning.IProvisioning
	var secrets []provisioning.OOBSecret

	capClient, err := hubclient.ConnectWithCapnpUDS(provisioning.ServiceName, runFolder)
	if err == nil {
		pc = capnpclient.NewProvisioningCapnpClient(capClient)
	}
	if err != nil {
		return err
	}
	manage, _ := pc.CapManageProvisioning(ctx, "hubcli")

	secrets = []provisioning.OOBSecret{
		{
			DeviceID:  deviceID,
			OobSecret: secret,
		},
	}
	err = manage.AddOOBSecrets(ctx, secrets)

	return err
}

// HandleApproveRequest
//
//	deviceID is the ID of the device to approve
func HandleApproveRequest(ctx context.Context, runFolder string, deviceID string) error {
	var pc provisioning.IProvisioning

	capClient, err := hubclient.ConnectWithCapnpUDS(provisioning.ServiceName, runFolder)
	if err == nil {
		pc = capnpclient.NewProvisioningCapnpClient(capClient)
		manage, _ := pc.CapManageProvisioning(ctx, "hubcli")
		err = manage.ApproveRequest(ctx, deviceID)
	}

	return err
}

func HandleGetApprovedRequests(ctx context.Context, runFolder string) error {
	var pc provisioning.IProvisioning
	var provStatus []provisioning.ProvisionStatus

	capClient, err := hubclient.ConnectWithCapnpUDS(provisioning.ServiceName, runFolder)
	if err == nil {
		pc = capnpclient.NewProvisioningCapnpClient(capClient)
		manage, _ := pc.CapManageProvisioning(ctx, "hubcli")
		provStatus, err = manage.GetApprovedRequests(ctx)
	}
	if err != nil {
		return err
	}

	fmt.Printf("Client ID              Request Time      Assigned\n")
	fmt.Printf("--------------------   ------------      --------\n")
	for _, provStatus := range provStatus {
		// a certificate is assigned when generated
		assigned := provStatus.ClientCertPEM != ""
		fmt.Printf("%20s  %s, %v\n",
			provStatus.DeviceID, provStatus.RequestTime, assigned)
	}

	return err
}

func HandleGetPendingRequests(ctx context.Context, runFolder string) error {
	var pc provisioning.IProvisioning
	var provStatus []provisioning.ProvisionStatus

	capClient, err := hubclient.ConnectWithCapnpUDS(provisioning.ServiceName, runFolder)
	if err == nil {
		pc = capnpclient.NewProvisioningCapnpClient(capClient)
		manage, _ := pc.CapManageProvisioning(ctx, "hubcli")
		provStatus, err = manage.GetPendingRequests(ctx)
	}
	if err != nil {
		return err
	}
	fmt.Printf("Client ID              Request Time\n")
	fmt.Printf("--------------------   ------------\n")
	for _, provStatus := range provStatus {
		// a certificate is assigned when generated
		fmt.Printf("%20s  %s\n",
			provStatus.DeviceID, provStatus.RequestTime)
	}

	return err
}
