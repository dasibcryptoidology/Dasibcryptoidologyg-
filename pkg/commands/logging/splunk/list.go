package splunk

import (
	"fmt"
	"io"

	"github.com/fastly/cli/pkg/cmd"
	fsterr "github.com/fastly/cli/pkg/errors"
	"github.com/fastly/cli/pkg/global"
	"github.com/fastly/cli/pkg/manifest"
	"github.com/fastly/cli/pkg/text"
	"github.com/fastly/go-fastly/v8/fastly"
)

// ListCommand calls the Fastly API to list Splunk logging endpoints.
type ListCommand struct {
	cmd.Base
	cmd.JSONOutput

	manifest       manifest.Data
	Input          fastly.ListSplunksInput
	serviceName    cmd.OptionalServiceNameID
	serviceVersion cmd.OptionalServiceVersion
}

// NewListCommand returns a usable command registered under the parent.
func NewListCommand(parent cmd.Registerer, g *global.Data, m manifest.Data) *ListCommand {
	c := ListCommand{
		Base: cmd.Base{
			Globals: g,
		},
		manifest: m,
	}
	c.CmdClause = parent.Command("list", "List Splunk endpoints on a Fastly service version")

	// Required.
	c.RegisterFlag(cmd.StringFlagOpts{
		Name:        cmd.FlagVersionName,
		Description: cmd.FlagVersionDesc,
		Dst:         &c.serviceVersion.Value,
		Required:    true,
	})

	// Optional.
	c.RegisterFlagBool(c.JSONFlag()) // --json
	c.RegisterFlag(cmd.StringFlagOpts{
		Name:        cmd.FlagServiceIDName,
		Description: cmd.FlagServiceIDDesc,
		Dst:         &c.manifest.Flag.ServiceID,
		Short:       's',
	})
	c.RegisterFlag(cmd.StringFlagOpts{
		Action:      c.serviceName.Set,
		Name:        cmd.FlagServiceName,
		Description: cmd.FlagServiceDesc,
		Dst:         &c.serviceName.Value,
	})
	return &c
}

// Exec invokes the application logic for the command.
func (c *ListCommand) Exec(_ io.Reader, out io.Writer) error {
	if c.Globals.Verbose() && c.JSONOutput.Enabled {
		return fsterr.ErrInvalidVerboseJSONCombo
	}

	serviceID, serviceVersion, err := cmd.ServiceDetails(cmd.ServiceDetailsOpts{
		AllowActiveLocked:  true,
		APIClient:          c.Globals.APIClient,
		Manifest:           c.manifest,
		Out:                out,
		ServiceNameFlag:    c.serviceName,
		ServiceVersionFlag: c.serviceVersion,
		VerboseMode:        c.Globals.Flags.Verbose,
	})
	if err != nil {
		c.Globals.ErrLog.AddWithContext(err, map[string]any{
			"Service ID":      serviceID,
			"Service Version": fsterr.ServiceVersion(serviceVersion),
		})
		return err
	}

	c.Input.ServiceID = serviceID
	c.Input.ServiceVersion = serviceVersion.Number

	o, err := c.Globals.APIClient.ListSplunks(&c.Input)
	if err != nil {
		c.Globals.ErrLog.Add(err)
		return err
	}

	if ok, err := c.WriteJSON(out, o); ok {
		return err
	}

	if !c.Globals.Verbose() {
		tw := text.NewTable(out)
		tw.AddHeader("SERVICE", "VERSION", "NAME")
		for _, splunk := range o {
			tw.AddLine(splunk.ServiceID, splunk.ServiceVersion, splunk.Name)
		}
		tw.Print()
		return nil
	}

	fmt.Fprintf(out, "Version: %d\n", c.Input.ServiceVersion)
	for i, splunk := range o {
		fmt.Fprintf(out, "\tSplunk %d/%d\n", i+1, len(o))
		fmt.Fprintf(out, "\t\tService ID: %s\n", splunk.ServiceID)
		fmt.Fprintf(out, "\t\tVersion: %d\n", splunk.ServiceVersion)
		fmt.Fprintf(out, "\t\tName: %s\n", splunk.Name)
		fmt.Fprintf(out, "\t\tURL: %s\n", splunk.URL)
		fmt.Fprintf(out, "\t\tToken: %s\n", splunk.Token)
		fmt.Fprintf(out, "\t\tTLS CA certificate: %s\n", splunk.TLSCACert)
		fmt.Fprintf(out, "\t\tTLS hostname: %s\n", splunk.TLSHostname)
		fmt.Fprintf(out, "\t\tTLS client certificate: %s\n", splunk.TLSClientCert)
		fmt.Fprintf(out, "\t\tTLS client key: %s\n", splunk.TLSClientKey)
		fmt.Fprintf(out, "\t\tFormat: %s\n", splunk.Format)
		fmt.Fprintf(out, "\t\tFormat version: %d\n", splunk.FormatVersion)
		fmt.Fprintf(out, "\t\tResponse condition: %s\n", splunk.ResponseCondition)
		fmt.Fprintf(out, "\t\tPlacement: %s\n", splunk.Placement)
	}
	fmt.Fprintln(out)

	return nil
}
