package inclient

import (
	"bytes"
	"context"
	"log"
	"strings"
	"time"

	"github.com/lxc/incus/v6/shared/api"
)

// InstanceUsers returns a list of users listed in an instance's /etc/passwd
// with UID >= 1000
func (c *Client) InstanceUsers(ctx context.Context, instance string) ([]string, error) {
	var outbuf bytes.Buffer
	var users []string
	retval, err := c.ExecBlind(
		[]string{instance, "awk", "-F:", "$3 >= 1000 && $1 != \"nobody\" {print $1}", "/etc/passwd"},
		[]string{},
		0,
		0,
		"/root",
		false,
		true,
		true,
		"auto",
		&outbuf,
	)
	if err != nil {
		log.Println(err)
	}
	if retval != 0 {
		log.Println(retval)

	}
	if err != nil {
		log.Println("error:", err.Error())
		log.Println("error:", outbuf.String())

		return []string{}, err
	}

	userString := outbuf.String()
	userString = strings.TrimSpace(userString)
	users = strings.Split(userString, "\n")
	return users, nil
}

// PrimaryUser returns the username of user 1000
func (c *Client) PrimaryUser(instance string) (string, error) {
	var outbuf bytes.Buffer
	retval, err := c.ExecBlind(
		[]string{instance, "awk", "-F:", "$3 == 1000 {print $1}", "/etc/passwd"},
		[]string{},
		0,
		0,
		"/root",
		false,
		true,
		true,
		"auto",
		&outbuf,
	)
	if err != nil {
		log.Println(err)
	}
	if retval != 0 {
		log.Println(retval)

	}
	if err != nil {
		log.Println("error:", err.Error())
		log.Println("error:", outbuf.String())

		return "", err
	}

	userString := outbuf.String()
	userString = strings.TrimSpace(userString)
	return userString, nil
}

// UserHome retrieves the $HOME folder location of a user in an instance
// as listed in /etc/passwd
func (c *Client) UserHome(instance string) (string, error) {
	var outbuf bytes.Buffer
	retval, err := c.ExecBlind(
		[]string{instance, "awk", "-F:", "$3 == 1000 {print $6}", "/etc/passwd"},
		[]string{},
		0,
		0,
		"/root",
		false,
		true,
		true,
		"auto",
		&outbuf,
	)
	if err != nil {
		log.Println(err)
	}
	if retval != 0 {
		log.Println(retval)

	}
	if err != nil {
		log.Println("error:", err.Error())
		log.Println("error:", outbuf.String())

		return "", err
	}

	userString := outbuf.String()
	userString = strings.TrimSpace(userString)
	return userString, nil
}

// Wait calls 'cloud-init status --wait' in an instance if it exists
// blocking until the --wait returns
func (c *Client) Wait(name string, vm bool, project string) ([]byte, error) {
	var outbuf bytes.Buffer
	if vm {
		time.Sleep(30 * time.Second)
	}
	retval, err := c.ExecBlind(
		[]string{name, "sh", "-c", "command -v cloud-init && cloud-init status --wait"},
		[]string{},
		0,
		0,
		"/root",
		false,
		true,
		true,
		"auto",
		&outbuf,
	)
	if err != nil {
		log.Println(err)
	}
	if retval != 0 {
		log.Println(retval)

	}
	if err != nil {
		log.Println("error:", err.Error())
		log.Println("error:", outbuf.String())

		return outbuf.Bytes(), err
	}

	return outbuf.Bytes(), nil
}

// InstanceState returns the state of an instance
func (c *Client) InstanceState(ctx context.Context, name string) (*api.InstanceFull, error) {
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return nil, err
	}

	// Get the full instance data.
	inst, _, err := d.GetInstanceFull(name)
	if err != nil {
		return nil, err
	}
	return inst, err
}

// StartInstance starts an existing instance
func (c *Client) StartInstance(ctx context.Context, name string) error {
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return err
	}

	op, err := d.UpdateInstanceState(name, api.InstanceStatePut{
		Action: "start",
	}, "")
	if err != nil {
		return err
	}
	return op.Wait()
}

// StopInstance stops an existing instance
func (c *Client) StopInstance(ctx context.Context, name string) error {
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return err
	}

	op, err := d.UpdateInstanceState(name, api.InstanceStatePut{
		Action: "stop",
	}, "")
	if err != nil {
		return err
	}
	return op.Wait()
}

// DeleteInstance deletes an existing instance
func (c *Client) DeleteInstance(ctx context.Context, name string) error {
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return err
	}

	op, err := d.DeleteInstance(name)
	if err != nil {
		return err
	}
	return op.Wait()
}
