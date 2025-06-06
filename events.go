package inclient

import (
	incus "github.com/lxc/incus/v6/client"
	config "github.com/lxc/incus/v6/shared/cliconfig"
)

// Listen returns an incus EventListener which can be polled for changes
// to the incus server
func (c *Client) Listen() (*incus.EventListener, error) {
	d, err := c.conf.GetInstanceServer(config.DefaultConfig().DefaultRemote)
	if err != nil {
		return nil, err
	}
	return d.GetEvents()
}
