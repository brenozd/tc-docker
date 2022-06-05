package docker

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/CodyGuo/glog"
	"github.com/brenozd/tc-docker/pkg/command"
)

type Veth struct {
	Device    string
	Ident     string
	LinkIdent string
}

func (c *Container) GetVeths(name, sandboxKey string) ([]string, error) {
	containerVeths, err := c.getContainerVeths(name, sandboxKey)
	if err != nil {
		return nil, err
	}
	hostVeths, err := c.getHostVeths()
	if err != nil {
		return nil, err
	}
	veths := []string{}
	for _, hv := range hostVeths {
		for _, cv := range containerVeths {
			if cv.Ident == hv.LinkIdent && cv.LinkIdent == hv.Ident {
				glog.Debugf("GetVeths found, container: %s, device: %s, veth: %+v", name, hv.Device, *cv)
				veths = append(veths, hv.Device)
			}
		}
	}
	if len(veths) == 0 {
		return nil, fmt.Errorf("container: %s, not found veth", name)
	}
	return veths, nil
}

func (c *Container) CreateIfb(name, veth string) (string, error) {
	id := strings.ReplaceAll(veth, "veth", "")
	ifb := fmt.Sprintf("ifb%s", id)

	// Create ifb to handle ingress traffic
	cmd := fmt.Sprintf("/usr/sbin/ip link add name %s type ifb", ifb)
	glog.Debug(cmd)
	out, err := command.CombinedOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
	}

	// Set ifb up
	cmd = fmt.Sprintf("/usr/sbin/ip link set dev %s up", ifb)
	glog.Debug(cmd)
	out, err = command.CombinedOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
	}

	// Crate ifb file and write ifb name to it
	// So we can delete it later
	f, errf := os.Create("/tmp/" + name)
	if errf != nil {
		return "", fmt.Errorf("failed to create ifb file %s, error: %v", name, errf)
	}

	_, errf = f.WriteString(ifb)
	if errf != nil {
		return "", fmt.Errorf("failed to write to ifb file %s, error: %v", name, errf)
	}

	return ifb, nil
}

func (c *Container) RemoveIfb(name string) error {
	ifb, err := ioutil.ReadFile("/tmp/" + name)
	if err != nil {
		os.Remove("/tmp/" + name)
		return fmt.Errorf("failed to read from ifb file %s, error: %v", name, err)
	}
	os.Remove("/tmp/" + name)

	// Set ifb down
	cmd := fmt.Sprintf("/usr/sbin/ip link set dev %s down", ifb)
	glog.Debug(cmd)
	out, err := command.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
	}

	// Delete ifb
	cmd = fmt.Sprintf("/usr/sbin/ip link del name %s", ifb)
	glog.Debug(cmd)
	out, err = command.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
	}

	glog.Debugf("RemoveIfb: %s", name)
	return nil
}

func (c *Container) RemoveVeth(name string) error {
	veth := "/var/run/docker/netns/" + name
	glog.Debugf("RemoveVeth: %s", veth)
	return os.Remove(veth)
}

func (c *Container) getHostVeths() ([]*Veth, error) {
	ipAddrCmd := "/usr/sbin/ip addr show type veth"
	glog.Debug(ipAddrCmd)
	out, err := command.CombinedOutput(ipAddrCmd)
	if err != nil {
		return nil, fmt.Errorf("out: %s, error: %v", out, err)
	}
	var veths []*Veth
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		b := scanner.Bytes()
		if !bytes.Contains(b, []byte("veth")) {
			continue
		}
		veth, err := parseVeth(b)
		if err != nil {
			glog.Errorf("getHostVeths, parseVeth: %v, veth: %s", err, b)
			continue
		}
		veths = append(veths, &veth)
	}
	return veths, nil
}

func (c *Container) getContainerVeths(name, sandboxKey string) ([]*Veth, error) {
	os.Remove("/var/run/docker/netns/" + name)
	if err := os.Symlink(sandboxKey, "/var/run/docker/netns/"+name); err != nil {
		return nil, err
	}
	ipAddrCmd := fmt.Sprintf("/usr/sbin/ip netns exec %s ip addr show ", name)
	glog.Debug(ipAddrCmd)
	out, err := command.CombinedOutput(ipAddrCmd)
	if err != nil {
		return nil, fmt.Errorf("out: %s, error: %v", out, err)
	}
	var veths []*Veth
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		b := scanner.Bytes()
		if !bytes.Contains(b, []byte("UP")) {
			continue
		}
		if bytes.Contains(b, []byte("LOOPBACK")) {
			continue
		}
		veth, err := parseVeth(b)
		if err != nil {
			glog.Errorf("getContainerVeths, parseVeth: %v, veth: %s", err, b)
			continue
		}
		veths = append(veths, &veth)
	}
	return veths, nil
}

func parseVeth(b []byte) (Veth, error) {
	fields := bytes.Split(b, []byte(":"))
	if len(fields) < 2 {
		return Veth{}, errors.New("not found")
	}
	ident := bytes.TrimSpace(fields[0])
	devices := bytes.Split(bytes.TrimSpace(fields[1]), []byte("@if"))
	if len(devices) < 2 {
		return Veth{}, errors.New("not found")

	}
	device := devices[0]
	link := devices[1]
	glog.Debugf("parseVeth, ident: %s, device: %s, link: %s", ident, device, link)
	return Veth{
		Device:    string(devices[0]),
		Ident:     string(ident),
		LinkIdent: string(devices[1]),
	}, nil
}
