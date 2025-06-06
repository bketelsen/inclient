package inclient

import (
	"context"
	"fmt"
	"os"
	"strings"

	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	config "github.com/lxc/incus/v6/shared/cliconfig"
)

// Client is an Incus client.
type Client struct {
	conf           *config.Config
	confPath       string
	currentProject string
	projectList    []string
	interactive    bool
}

// NewClient returns a new Client based on the configuration in
// either INCUS_ environment variables or '.config/incus/config.yml'
func NewClient(conf *config.Config) (*Client, error) {

	return &Client{
		conf: conf,
		//confPath: confPath,
	}, nil
}

// RefreshProjectList pulls the project list from Incus
// and sets the value in the struct
func (c *Client) RefreshProjectList() ([]string, error) {
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return []string{}, err
	}
	c.projectList, err = d.GetProjectNames()
	if err != nil {
		return []string{}, err
	}
	return c.projectList, nil
}

// Instances returns the list of Instances from Incus
func (c *Client) Instances(ctx context.Context) ([]api.InstanceFull, error) {
	var ii []api.InstanceFull
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return ii, err
	}

	return d.GetInstancesFull(api.InstanceTypeAny)
}

// Instance returns a single instance from Incus
func (c *Client) Instance(ctx context.Context, name string) (*api.InstanceFull, string, error) {
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return nil, "", err
	}

	return d.GetInstanceFull(name)
}

// Profiles returns the full list of Profiles from Incus
func (c *Client) Profiles(ctx context.Context) ([]api.Profile, error) {
	var ii []api.Profile
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return ii, err
	}

	return d.GetProfiles()
}

// Profile returns a single profile from Incus
func (c *Client) Profile(ctx context.Context, name string) (*api.Profile, error) {
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return &api.Profile{}, err
	}

	p, _, err := d.GetProfile(name)
	return p, err
}

// Networks returns a list of Networks from Incus
func (c *Client) Networks(ctx context.Context) ([]api.Network, error) {
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return []api.Network{}, err
	}

	n, err := d.GetNetworks()
	return n, err
}

// Profile names returns profile names as a string slice
func (c *Client) ProfileNames(ctx context.Context) ([]string, error) {
	var profiles []string
	pp, err := c.Profiles(ctx)
	if err != nil {
		return profiles, err
	}
	for _, p := range pp {
		profiles = append(profiles, p.Name)
	}
	return profiles, nil
}

// ProfileCreate adds a new Profile to Incus
func (c *Client) ProfileCreate(ctx context.Context, put api.ProfilesPost) error {
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return err
	}
	err = d.CreateProfile(put)
	if err != nil {
		return err
	}
	return nil

}

// ProjectList returns the Client's projectList member
func (c *Client) ProjectList() []string {
	return c.projectList
}

func (c *Client) SetProject(id uint) {
	c.currentProject = c.projectList[id]
	c.conf.ProjectOverride = c.projectList[id]
}

// StorageList returns the list of Storage Pools from Incus
func (c *Client) StorageList(ctx context.Context) ([]api.StoragePool, error) {
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return []api.StoragePool{}, err
	}
	sp, err := d.GetStoragePools()
	if err != nil {
		return []api.StoragePool{}, err
	}
	return sp, nil
}

// Launch creates and starts a new Instance
func (c *Client) Launch(image string, name string, profiles []string, extraConfigs map[string]string, deviceOverrides map[string]map[string]string, netname string, vm, launch bool) error {
	// Call the matching code from init
	// skip userdata for now
	d, _, err := c.create(image, name, profiles, extraConfigs, deviceOverrides, netname, vm, launch)
	if err != nil {
		return err
	}
	// Check if the instance was started by the server.
	if d.HasExtension("instance_create_start") {
		return nil
	}

	return nil
}

func (c *Client) create(image string, name string, requestedProfiles []string, extraConfigs map[string]string, deviceOverrides map[string]map[string]string, netName string, vm, launch bool) (incus.InstanceServer, string, error) {

	var remote string
	var iremote string
	var err error
	var stdinData api.InstancePut
	var devicesMap map[string]map[string]string
	//var configMap map[string]string
	var profiles []string
	// if !strings.HasSuffix(image, "cloud") {
	// 	image = image + "/cloud"
	// }

	if !strings.HasPrefix(image, "images:") {
		image = "images:" + image
	}
	iremote, image, err = c.conf.ParseRemote(image)
	if err != nil {
		return nil, "", err
	}
	remote, name, err = c.conf.ParseRemote(name)
	if err != nil {
		return nil, "", err
	}

	d, err := c.conf.GetInstanceServer(remote)
	if err != nil {
		return nil, "", err
	}

	if d.HasExtension("instance_create_start") && launch {
		if name == "" {
			fmt.Printf("Launching the instance" + "\n")
		} else {
			fmt.Printf("Launching %s"+"\n", name)
		}
	} else {
		if name == "" {
			fmt.Printf("Creating the instance" + "\n")
		} else {
			fmt.Printf("Creating %s"+"\n", name)
		}
	}

	//profiles = append(profiles, "default")
	profiles = append(profiles, requestedProfiles...)

	devicesMap = map[string]map[string]string{}

	if netName != "" {
		network, _, err := d.GetNetwork(netName)
		if err != nil {
			return nil, "", fmt.Errorf("failed loading network %q: %w", network.Name, err)
		}

		// Prepare the instance's NIC device entry.
		var device map[string]string

		if network.Managed && d.HasExtension("instance_nic_network") {
			// If network is managed, use the network property rather than nictype, so that the
			// network's inherited properties are loaded into the NIC when started.
			device = map[string]string{
				"name":    "eth0",
				"type":    "nic",
				"network": network.Name,
			}
		} else {
			// If network is unmanaged default to using a macvlan connected to the specified interface.
			device = map[string]string{
				"name":    "eth0",
				"type":    "nic",
				"nictype": "macvlan",
				"parent":  network.Name,
			}

			if network.Type == "bridge" {
				// If the network type is an unmanaged bridge, use bridged NIC type.
				device["nictype"] = "bridged"
			}
		}

		devicesMap["eth0"] = device
	}

	// Decide whether we are creating a container or a virtual machine.
	instanceDBType := api.InstanceTypeContainer
	if vm {
		instanceDBType = api.InstanceTypeVM
	}

	// Setup instance creation request
	req := api.InstancesPost{
		Name: name,
		//InstanceType: c.flagType, #TODO with vms
		Type:  instanceDBType,
		Start: launch,
	}

	req.Config = extraConfigs
	req.Ephemeral = false
	req.Description = stdinData.Description

	req.Profiles = profiles

	// Check to see if any of the overridden devices are for devices that are not yet defined in the
	// local devices (and thus maybe expected to be coming from profiles).
	profileDevices := make(map[string]map[string]string)
	needProfileExpansion := false
	for deviceName := range deviceOverrides {
		_, isLocalDevice := devicesMap[deviceName]
		if !isLocalDevice {
			needProfileExpansion = true
			break
		}
	}

	// If there are device overrides that are expected to be applied to profile devices then load the profiles
	// that would be applied server-side.
	if needProfileExpansion {
		// If the list of profiles is empty then the default profile would be applied on the server side.
		serverSideProfiles := req.Profiles
		if len(serverSideProfiles) == 0 {
			serverSideProfiles = []string{"default"}
		}

		// Get the effective expanded devices by overlaying each profile's devices in order.
		for _, profileName := range serverSideProfiles {
			profile, _, err := d.GetProfile(profileName)
			if err != nil {
				return nil, "", fmt.Errorf("failed loading profile %q for device override: %w", profileName, err)
			}

			for k, v := range profile.Devices {
				profileDevices[k] = v
			}
		}
	}

	// Apply device overrides.
	for deviceName := range deviceOverrides {
		_, isLocalDevice := devicesMap[deviceName]
		if isLocalDevice {
			// Apply overrides to local device.
			for k, v := range deviceOverrides[deviceName] {
				devicesMap[deviceName][k] = v
			}
		} else {
			// Check device exists in expanded profile devices.
			profileDeviceConfig, found := profileDevices[deviceName]
			if !found {
				return nil, "", fmt.Errorf("cannot override config for device %q: Device not found in profile devices", deviceName)
			}

			for k, v := range deviceOverrides[deviceName] {
				profileDeviceConfig[k] = v
			}

			// Add device to local devices.
			devicesMap[deviceName] = profileDeviceConfig
		}
	}

	req.Devices = devicesMap

	// Get the image server and image info
	iremote, image = guessImage(c.conf, d, remote, iremote, image)

	// Deal with the default image
	if image == "" {
		image = "default"
	}

	imgRemote, imgInfo, err := getImgInfo(d, c.conf, iremote, remote, image, &req.Source)
	if err != nil {
		return nil, "", err
	}

	if c.conf.Remotes[iremote].Protocol != "simplestreams" {
		if imgInfo.Type != "virtual-machine" && vm {
			return nil, "", fmt.Errorf("asked for a VM but image is of type container")
		}

		req.Type = api.InstanceType(imgInfo.Type)
	}

	// Create the instance
	op, err := d.CreateInstanceFromImage(imgRemote, *imgInfo, req)
	if err != nil {
		return nil, "", err
	}

	err = op.Wait()
	if err != nil {
		return nil, "", err
	}
	// Validate the network setup
	c.checkNetwork(d, name)

	// // wait for cloud init
	// slog.Debug("wait for cloud-init", slog.String("instance", name))
	// out, err := c.Wait(name, vm, c.conf.ProjectOverride)
	// if err != nil {
	// 	return d, name, fmt.Errorf(string(out))
	// }

	return d, name, nil
}

// AddDeviceToInstance adds a device to an existing Incus instance
func (c *Client) AddDeviceToInstance(ctx context.Context, instance, devname string, device map[string]string) error {
	var remote string
	var err error

	remote, _, err = c.conf.ParseRemote(instance)
	if err != nil {
		return err
	}

	d, err := c.conf.GetInstanceServer(remote)
	if err != nil {
		return err
	}

	inst, etag, err := d.GetInstance(instance)
	if err != nil {
		return err
	}

	_, ok := inst.Devices[devname]
	if ok {
		return fmt.Errorf("the device already exists")
	}

	inst.Devices[devname] = device

	op, err := d.UpdateInstance(instance, inst.Writable(), etag)
	if err != nil {
		return err
	}

	return op.Wait()

}

// guessImage checks that the image name (provided by the user) is correct given an instance remote and image remote.
func guessImage(conf *config.Config, d incus.InstanceServer, instRemote string, imgRemote string, imageRef string) (string, string) {
	if instRemote != imgRemote {
		return imgRemote, imageRef
	}

	fields := strings.SplitN(imageRef, "/", 2)
	_, ok := conf.Remotes[fields[0]]
	if !ok {
		return imgRemote, imageRef
	}

	_, _, err := d.GetImageAlias(imageRef)
	if err == nil {
		return imgRemote, imageRef
	}

	_, _, err = d.GetImage(imageRef)
	if err == nil {
		return imgRemote, imageRef
	}

	if len(fields) == 1 {
		fmt.Fprintf(os.Stderr, "The local image '%q' couldn't be found, trying '%q:' instead.\n", imageRef, fields[0])
		return fields[0], "default"
	}

	fmt.Fprintf(os.Stderr, "The local image '%q' couldn't be found, trying '%q:%q' instead.\n", imageRef, fields[0], fields[1])
	return fields[0], fields[1]
}

// getImgInfo returns an image server and image info for the given image name (given by a user)
// an image remote and an instance remote.
func getImgInfo(d incus.InstanceServer, conf *config.Config, imgRemote string, instRemote string, imageRef string, source *api.InstanceSource) (incus.ImageServer, *api.Image, error) {
	var imgRemoteServer incus.ImageServer
	var imgInfo *api.Image
	var err error

	// Connect to the image server
	if imgRemote == instRemote {
		imgRemoteServer = d
	} else {
		imgRemoteServer, err = conf.GetImageServer(imgRemote)
		if err != nil {
			return nil, nil, err
		}
	}

	// Optimisation for simplestreams
	if conf.Remotes[imgRemote].Protocol == "simplestreams" {
		imgInfo = &api.Image{}
		imgInfo.Fingerprint = imageRef
		imgInfo.Public = true
		source.Alias = imageRef
	} else {
		// Attempt to resolve an image alias
		alias, _, err := imgRemoteServer.GetImageAlias(imageRef)
		if err == nil {
			source.Alias = imageRef
			imageRef = alias.Target
		}

		// Get the image info
		imgInfo, _, err = imgRemoteServer.GetImage(imageRef)
		if err != nil {
			return nil, nil, err
		}
	}

	return imgRemoteServer, imgInfo, nil
}

// checkNetwork runs network checks on an instance
func (c *Client) checkNetwork(d incus.InstanceServer, name string) {
	ct, _, err := d.GetInstance(name)
	if err != nil {
		return
	}

	for _, d := range ct.ExpandedDevices {
		if d["type"] == "nic" {
			return
		}
	}

	fmt.Fprintf(os.Stderr, "\nThe instance you are starting doesn't have any network attached to it.\n")
	fmt.Fprintf(os.Stderr, "  To create a new network, use: incus network create\n")
	fmt.Fprintf(os.Stderr, "  To attach a network to an instance, use: incus network attach\n\n")
}

// Instance returns a single instance from Incus
func (c *Client) IsTrueNAS(ctx context.Context) (bool, error) {
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return false, err
	}
	return IsTrueNAS(d)
}
func (c *Client) RemoveConfig(ctx context.Context, name string, configName string) error {
	d, err := c.conf.GetInstanceServer(c.conf.DefaultRemote)
	if err != nil {
		return err
	}
	inst, etag, err := d.GetInstance(name)
	if err != nil {
		return err
	}

	_, ok := inst.Config[configName]
	if ok {
		return fmt.Errorf("the configuration does not exist")
	}

	inst.Config[configName] = ""

	op, err := d.UpdateInstance(name, inst.Writable(), etag)
	if err != nil {
		return err
	}

	return op.Wait()

}
func IsTrueNAS(d incus.InstanceServer) (bool, error) {
	s, _, err := d.GetServer()
	if err != nil {
		return false, err
	}
	if strings.Contains(s.Environment.KernelVersion, "truenas") {
		return true, nil
	}
	return false, nil
}
