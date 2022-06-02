package tc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/CodyGuo/glog"
	"github.com/brenozd/tc-docker/internal/docker"
	"github.com/brenozd/tc-docker/pkg/command"
)

var (
	ErrTcNotFound = errors.New("RTNETLINK answers: No such file or directory")
)

func SetTC(container *docker.Container) error {
	// Delete any existing root qdisc in container.Veth
	cmd := fmt.Sprintf("/usr/sbin/tc qdisc del dev %s root", container.Veth)
	glog.Debug(cmd)
	out, err := command.CombinedOutput(cmd)
	if err != nil {
		if strings.TrimSpace(string(out)) != ErrTcNotFound.Error() {
			return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
		}
	}

	// Create HTB qdisc to limit egress traffic in container.Veth
	cmd = fmt.Sprintf("/usr/sbin/tc qdisc add dev %s root handle 1: htb r2q 1 default 2", container.Veth)
	glog.Debug(cmd)
	out, err = command.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
	}

	// Set egress bandwidth limit
	cmd = fmt.Sprintf("/usr/sbin/tc class add dev %s parent 1: classid 1:2 htb rate %s ceil %s prio 2", container.Veth, container.UploadRate, container.UploadCeil)
	glog.Debug(cmd)
	out, err = command.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
	}

	netemFlags := "delay " + container.LatencyDelay

	if container.LatencyDelay != "0ms" && container.LatencyVariation != "" {
		netemFlags += " " + container.LatencyVariation
		if container.LatencyCorrelation != "" {
			netemFlags += " " + container.LatencyCorrelation
		}
	}

	if container.LossProbability != "" {
		netemFlags += " loss " + container.LossProbability
		if container.LossCorrelation != "" {
			netemFlags += " " + container.LossCorrelation
		}
	}

	if container.PacketDuplication != "" {
		netemFlags += " duplicate " + container.PacketDuplication
	}

	if container.PacketCorruption != "" {
		netemFlags += " corrupt " + container.PacketCorruption
	}

	if container.PacketReordering != "" {
		netemFlags += " reorder " + container.PacketReordering
	}

	// Set netem
	cmd = fmt.Sprintf("/usr/sbin/tc qdisc add dev %s parent 1:2 handle 10:0 netem %s", container.Veth, netemFlags)
	glog.Debug(cmd)
	out, err = command.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
	}

	// Apply to all traffic going through container.Veth
	cmd = fmt.Sprintf("/usr/sbin/tc filter add dev %s parent 1:0 matchall flowid 1:2", container.Veth)
	glog.Debug(cmd)
	out, err = command.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
	}

	if container.Ifb == "" {
		return fmt.Errorf("cannot create container.Ifb interface to limit ingress traffic")
	}

	// Delete any existing root qdisc in container.Veth
	cmd = fmt.Sprintf("/usr/sbin/tc qdisc del dev %s root", container.Ifb)
	glog.Debug(cmd)
	out, err = command.CombinedOutput(cmd)
	if err != nil {
		if strings.TrimSpace(string(out)) != ErrTcNotFound.Error() {
			return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
		}
	}

	// Create HTB qdisc to limit egress traffic in container.Veth
	cmd = fmt.Sprintf("/usr/sbin/tc qdisc add dev %s root handle 1: htb r2q 1", container.Ifb)
	glog.Debug(cmd)
	out, err = command.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
	}

	// Set egress bandwidth limit
	cmd = fmt.Sprintf("/usr/sbin/tc class add dev %s parent 1: classid 1:1 htb rate %s ceil %s", container.Ifb, container.DownloadRate, container.DownloadCeil)
	glog.Debug(cmd)
	out, err = command.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
	}

	// Apply to all traffic going through container.Veth
	cmd = fmt.Sprintf("/usr/sbin/tc filter add dev %s parent 1: matchall flowid 1:1", container.Ifb)
	glog.Debug(cmd)
	out, err = command.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
	}

	// Delete ingress qdisc in container.Veth
	cmd = fmt.Sprintf("/usr/sbin/tc qdisc del dev %s ingress", container.Veth)
	glog.Debug(cmd)
	out, err = command.CombinedOutput(cmd)
	if err != nil {
		if strings.TrimSpace(string(out)) != ErrTcNotFound.Error() {
			return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
		}
	}

	// Create ingress qdisc in container.Veth
	cmd = fmt.Sprintf("/usr/sbin/tc qdisc add dev %s ingress", container.Veth)
	glog.Debug(cmd)
	out, err = command.CombinedOutput(cmd)
	if err != nil {
		if strings.TrimSpace(string(out)) != ErrTcNotFound.Error() {
			return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
		}
	}

	// Mirror every ingress traffic from eth0 to container.Ifb0
	cmd = fmt.Sprintf("/usr/sbin/tc filter add dev %s ingress matchall action mirred egress redirect dev %s", container.Veth, container.Ifb)
	glog.Debug(cmd)
	out, err = command.CombinedOutput(cmd)
	if err != nil {
		if strings.TrimSpace(string(out)) != ErrTcNotFound.Error() {
			return fmt.Errorf("cmd: %s, out: %s, error: %v", cmd, out, err)
		}
	}

	return nil
}

func GetTcString(c *docker.Container) string {
	tcString := fmt.Sprintf("container: %s, id: %s, veth: %s, ifb: %s, download rate: %s, download ceil: %s, upload rate %s, upload ceil %s",
		c.Name, c.ID, c.Veth, c.Ifb,
		c.DownloadRate, c.DownloadCeil,
		c.UploadRate, c.UploadCeil)

	if c.LatencyDelay != "0ms" {
		tcString += fmt.Sprintf(", latency delay: %s", c.LatencyDelay)
		if c.LatencyVariation != "" {
			tcString += fmt.Sprintf(", latency variation: %s", c.LatencyVariation)
		}
		if c.LatencyCorrelation != "" {
			tcString += fmt.Sprintf(", latency correlation: %s", c.LatencyCorrelation)
		}
	}

	if c.LossProbability != "" {
		tcString += fmt.Sprintf(", loss probability: %s", c.LossProbability)
		if c.LossCorrelation != "" {
			tcString += fmt.Sprintf(", loss correlation: %s", c.LossCorrelation)
		}
	}

	if c.PacketDuplication != "" {
		tcString += fmt.Sprintf(", packet duplication: %s", c.PacketDuplication)
	}

	if c.PacketCorruption != "" {
		tcString += fmt.Sprintf(", packet corruption: %s", c.PacketCorruption)
	}

	if c.PacketReordering != "" {
		tcString += fmt.Sprintf(", packet reordeing: %s", c.PacketReordering)
	}

	return tcString
}
