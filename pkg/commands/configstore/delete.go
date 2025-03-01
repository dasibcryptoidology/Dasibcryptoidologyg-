package configstore

import (
	"io"

	"github.com/fastly/go-fastly/v8/fastly"

	"github.com/fastly/cli/pkg/cmd"
	fsterr "github.com/fastly/cli/pkg/errors"
	"github.com/fastly/cli/pkg/global"
	"github.com/fastly/cli/pkg/manifest"
	"github.com/fastly/cli/pkg/text"
)

// NewDeleteCommand returns a usable command registered under the parent.
func NewDeleteCommand(parent cmd.Registerer, g *global.Data, m manifest.Data) *DeleteCommand {
	c := DeleteCommand{
		Base: cmd.Base{
			Globals: g,
		},
		manifest: m,
	}

	c.CmdClause = parent.Command("delete", "Delete a config store")

	// Required.
	c.RegisterFlag(cmd.StoreIDFlag(&c.input.ID)) // --store-id

	// Optional.
	c.RegisterFlagBool(c.JSONFlag()) // --json

	return &c
}

// DeleteCommand calls the Fastly API to delete an appropriate resource.
type DeleteCommand struct {
	cmd.Base
	cmd.JSONOutput

	input    fastly.DeleteConfigStoreInput
	manifest manifest.Data
}

// Exec invokes the application logic for the command.
func (c *DeleteCommand) Exec(_ io.Reader, out io.Writer) error {
	if c.Globals.Verbose() && c.JSONOutput.Enabled {
		return fsterr.ErrInvalidVerboseJSONCombo
	}

	err := c.Globals.APIClient.DeleteConfigStore(&c.input)
	if err != nil {
		c.Globals.ErrLog.Add(err)
		return err
	}

	if c.JSONOutput.Enabled {
		o := struct {
			ID      string `json:"id"`
			Deleted bool   `json:"deleted"`
		}{
			c.input.ID,
			true,
		}
		_, err := c.WriteJSON(out, o)
		return err
	}

	text.Success(out, "Deleted Config Store '%s'", c.input.ID)

	return nil
}
