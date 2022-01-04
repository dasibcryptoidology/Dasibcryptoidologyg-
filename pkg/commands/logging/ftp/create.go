package ftp

import (
	"fmt"
	"io"

	"github.com/fastly/cli/pkg/cmd"
	"github.com/fastly/cli/pkg/config"
	"github.com/fastly/cli/pkg/errors"
	"github.com/fastly/cli/pkg/manifest"
	"github.com/fastly/cli/pkg/text"
	"github.com/fastly/go-fastly/v5/fastly"
)

// CreateCommand calls the Fastly API to create an FTP logging endpoint.
type CreateCommand struct {
	cmd.Base
	Manifest manifest.Data

	// required
	EndpointName   string // Can't shadow cmd.Base method Name().
	Address        string
	Username       string
	Password       string
	ServiceName    cmd.OptionalServiceNameID
	ServiceVersion cmd.OptionalServiceVersion

	// optional
	AutoClone         cmd.OptionalAutoClone
	Port              cmd.OptionalUint
	Path              cmd.OptionalString
	Period            cmd.OptionalUint
	GzipLevel         cmd.OptionalUint8
	Format            cmd.OptionalString
	FormatVersion     cmd.OptionalUint
	ResponseCondition cmd.OptionalString
	TimestampFormat   cmd.OptionalString
	Placement         cmd.OptionalString
	CompressionCodec  cmd.OptionalString
}

// NewCreateCommand returns a usable command registered under the parent.
func NewCreateCommand(parent cmd.Registerer, globals *config.Data, data manifest.Data) *CreateCommand {
	var c CreateCommand
	c.Globals = globals
	c.Manifest = data
	c.CmdClause = parent.Command("create", "Create an FTP logging endpoint on a Fastly service version").Alias("add")
	c.CmdClause.Flag("name", "The name of the FTP logging object. Used as a primary key for API access").Short('n').Required().StringVar(&c.EndpointName)
	c.RegisterFlag(cmd.StringFlagOpts{
		Name:        cmd.FlagVersionName,
		Description: cmd.FlagVersionDesc,
		Dst:         &c.ServiceVersion.Value,
		Required:    true,
	})
	c.RegisterAutoCloneFlag(cmd.AutoCloneFlagOpts{
		Action: c.AutoClone.Set,
		Dst:    &c.AutoClone.Value,
	})
	c.CmdClause.Flag("address", "An hostname or IPv4 address").Required().StringVar(&c.Address)
	c.CmdClause.Flag("user", "The username for the server (can be anonymous)").Required().StringVar(&c.Username)
	c.CmdClause.Flag("password", "The password for the server (for anonymous use an email address)").Required().StringVar(&c.Password)
	c.RegisterFlag(cmd.StringFlagOpts{
		Name:        cmd.FlagServiceIDName,
		Description: cmd.FlagServiceIDDesc,
		Dst:         &c.Manifest.Flag.ServiceID,
		Short:       's',
	})
	c.RegisterFlag(cmd.StringFlagOpts{
		Action:      c.ServiceName.Set,
		Name:        cmd.FlagServiceName,
		Description: cmd.FlagServiceDesc,
		Dst:         &c.ServiceName.Value,
	})
	c.CmdClause.Flag("port", "The port number").Action(c.Port.Set).UintVar(&c.Port.Value)
	c.CmdClause.Flag("path", "The path to upload log files to. If the path ends in / then it is treated as a directory").Action(c.Path.Set).StringVar(&c.Path.Value)
	c.CmdClause.Flag("period", "How frequently log files are finalized so they can be available for reading (in seconds, default 3600)").Action(c.Period.Set).UintVar(&c.Period.Value)
	c.CmdClause.Flag("gzip-level", "What level of GZIP encoding to have when dumping logs (default 0, no compression)").Action(c.GzipLevel.Set).Uint8Var(&c.GzipLevel.Value)
	c.CmdClause.Flag("format", "Apache style log formatting").Action(c.Format.Set).StringVar(&c.Format.Value)
	c.CmdClause.Flag("format-version", "The version of the custom logging format used for the configured endpoint. Can be either 2 (default) or 1").Action(c.FormatVersion.Set).UintVar(&c.FormatVersion.Value)
	c.CmdClause.Flag("response-condition", "The name of an existing condition in the configured endpoint, or leave blank to always execute").Action(c.ResponseCondition.Set).StringVar(&c.ResponseCondition.Value)
	c.CmdClause.Flag("timestamp-format", `strftime specified timestamp formatting (default "%Y-%m-%dT%H:%M:%S.000")`).Action(c.TimestampFormat.Set).StringVar(&c.TimestampFormat.Value)
	c.CmdClause.Flag("placement", "Where in the generated VCL the logging call should be placed, overriding any format_version default. Can be none or waf_debug").Action(c.Placement.Set).StringVar(&c.Placement.Value)
	c.CmdClause.Flag("compression-codec", `The codec used for compression of your logs. Valid values are zstd, snappy, and gzip. If the specified codec is "gzip", gzip_level will default to 3. To specify a different level, leave compression_codec blank and explicitly set the level using gzip_level. Specifying both compression_codec and gzip_level in the same API request will result in an error.`).Action(c.CompressionCodec.Set).StringVar(&c.CompressionCodec.Value)
	return &c
}

// ConstructInput transforms values parsed from CLI flags into an object to be used by the API client library.
func (c *CreateCommand) ConstructInput(serviceID string, serviceVersion int) (*fastly.CreateFTPInput, error) {
	var input fastly.CreateFTPInput

	input.ServiceID = serviceID
	input.ServiceVersion = serviceVersion
	input.Name = c.EndpointName
	input.Address = c.Address
	input.Username = c.Username
	input.Password = c.Password

	// The following blocks enforces the mutual exclusivity of the
	// CompressionCodec and GzipLevel flags.
	if c.CompressionCodec.WasSet && c.GzipLevel.WasSet {
		return nil, fmt.Errorf("error parsing arguments: the --compression-codec flag is mutually exclusive with the --gzip-level flag")
	}

	if c.Port.WasSet {
		input.Port = c.Port.Value
	}

	if c.Path.WasSet {
		input.Path = c.Path.Value
	}

	if c.Period.WasSet {
		input.Period = c.Period.Value
	}

	if c.Format.WasSet {
		input.Format = c.Format.Value
	}

	if c.FormatVersion.WasSet {
		input.FormatVersion = c.FormatVersion.Value
	}

	if c.GzipLevel.WasSet {
		input.GzipLevel = c.GzipLevel.Value
	}

	if c.ResponseCondition.WasSet {
		input.ResponseCondition = c.ResponseCondition.Value
	}

	if c.TimestampFormat.WasSet {
		input.TimestampFormat = c.TimestampFormat.Value
	}

	if c.Placement.WasSet {
		input.Placement = c.Placement.Value
	}

	if c.CompressionCodec.WasSet {
		input.CompressionCodec = c.CompressionCodec.Value
	}

	return &input, nil
}

// Exec invokes the application logic for the command.
func (c *CreateCommand) Exec(in io.Reader, out io.Writer) error {
	serviceID, serviceVersion, err := cmd.ServiceDetails(cmd.ServiceDetailsOpts{
		AutoCloneFlag:      c.AutoClone,
		Client:             c.Globals.Client,
		Manifest:           c.Manifest,
		Out:                out,
		ServiceNameFlag:    c.ServiceName,
		ServiceVersionFlag: c.ServiceVersion,
		VerboseMode:        c.Globals.Flag.Verbose,
	})
	if err != nil {
		c.Globals.ErrLog.AddWithContext(err, map[string]interface{}{
			"Service ID":      serviceID,
			"Service Version": errors.ServiceVersion(serviceVersion),
		})
		return err
	}

	input, err := c.ConstructInput(serviceID, serviceVersion.Number)
	if err != nil {
		c.Globals.ErrLog.Add(err)
		return err
	}

	d, err := c.Globals.Client.CreateFTP(input)
	if err != nil {
		c.Globals.ErrLog.Add(err)
		return err
	}

	text.Success(out, "Created FTP logging endpoint %s (service %s version %d)", d.Name, d.ServiceID, d.ServiceVersion)
	return nil
}
