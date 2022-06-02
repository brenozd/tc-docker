package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/CodyGuo/glog"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type Container struct {
	ctx                context.Context
	dc                 *client.Client
	event              EventHandler
	ID                 string
	Name               string
	Veth               string
	Ifb                string
	DownloadRate       string
	DownloadCeil       string
	UploadRate         string
	UploadCeil         string
	LatencyDelay       string
	LatencyVariation   string
	LatencyCorrelation string
	LossProbability    string
	LossCorrelation    string
	PacketDuplication  string
	PacketCorruption   string
	PacketReordering   string
}

func NewContainer(ctx context.Context, dc *client.Client) *Container {
	c := &Container{ctx: ctx, dc: dc}
	go c.eventWatch()
	return c
}

func (c *Container) GetRunningList() ([]*Container, error) {
	f := filters.NewArgs()
	f.Add("label", "org.label-schema.tc.enabled=1")
	f.Add("status", "running")
	containerList, err := c.dc.ContainerList(c.ctx, types.ContainerListOptions{Filters: f})
	if err != nil {
		return nil, fmt.Errorf("ContainerList error: %v", err)
	}

	var containers []*Container
	for _, container := range containerList {
		name, err := c.getName(container.ID)
		if err != nil {
			return nil, fmt.Errorf("getName error: %v", err)
		}
		sandboxKey, err := c.getSandboxKey(container.ID)
		if err != nil {
			return nil, fmt.Errorf("getSandboxKey error: %v", err)
		}
		veths, err := c.GetVeths(name, sandboxKey)
		if err != nil {
			return nil, fmt.Errorf("GetRunningList.getVeths error: %v", err)
		}
		downloadRate, downloadCeil, uploadRate, uploadCeil,
			latencyDelay, latencyVariation, latencyCorrelation,
			lossProbability, lossCorrelation,
			packetDuplication, packetCorruption, packetReordering := c.getLabelTC(container.Labels)
		for _, veth := range veths {
			ifb, err := c.CreateIfb(name, veth)
			if err != nil {
				glog.Errorf("cannot create ifb for container %s, tc will not be running on this one", name)
			}
			containers = append(containers, &Container{
				ID:                 container.ID[:12],
				Name:               name,
				Veth:               veth,
				Ifb:                ifb,
				DownloadRate:       downloadRate,
				DownloadCeil:       downloadCeil,
				UploadRate:         uploadRate,
				UploadCeil:         uploadCeil,
				LatencyDelay:       latencyDelay,
				LatencyVariation:   latencyVariation,
				LatencyCorrelation: latencyCorrelation,
				LossProbability:    lossProbability,
				LossCorrelation:    lossCorrelation,
				PacketDuplication:  packetDuplication,
				PacketCorruption:   packetCorruption,
				PacketReordering:   packetReordering,
			})
		}
	}
	return containers, nil
}

func (c *Container) getName(containerID string) (string, error) {
	cJson, err := c.dc.ContainerInspect(c.ctx, containerID)
	if err != nil {
		return "", err
	}
	return strings.TrimLeft(cJson.Name, "/"), nil
}

func (c *Container) getSandboxKey(containerID string) (string, error) {
	cJson, err := c.dc.ContainerInspect(c.ctx, containerID)
	if err != nil {
		return "", err
	}
	return cJson.NetworkSettings.SandboxKey, nil
}

func (c *Container) getLabelTC(labels map[string]string) (string, string, string, string, string, string, string, string, string, string, string, string) {
	uploadRate, hasUploadRate := labels["org.label-schema.tc.upload.rate"]
	uploadCeil, hasUploadCeil := labels["org.label-schema.tc.upload.ceil"]
	downloadRate, hasDownloadRate := labels["org.label-schema.tc.download.rate"]
	downloadCeil, hasDownloadCeil := labels["org.label-schema.tc.download.ceil"]
	latencyDelay, hasLatencyDelay := labels["org.label-schema.tc.latency.delay"]
	latencyVariation, _ := labels["org.label-schema.tc.latency.variation"]
	latencyCorrelation, _ := labels["org.label-schema.tc.latency.correlation"]
	lossProbability, _ := labels["org.label-schema.tc.loss.probability"]
	lossCorrelation, _ := labels["org.label-schema.tc.loss.correlation"]
	packetDuplication, _ := labels["org.label-schema.tc.packet.duplication"]
	packetCorruption, _ := labels["org.label-schema.tc.packet.corruption"]
	packetReordering, _ := labels["org.label-schema.tc.packet.reordering"]

	// Check for empty upload labels
	if !hasUploadRate && !hasUploadCeil {
		uploadRate = "10000mbps"
		uploadCeil = "10000mbps"
	} else if hasUploadRate && !hasUploadCeil {
		uploadCeil = uploadRate
	} else if hasUploadCeil && !hasUploadRate {
		uploadRate = uploadCeil
	}

	// Check for empty download labels
	if !hasDownloadRate && !hasDownloadCeil {
		downloadRate = "10000mbps"
		downloadCeil = "10000mbps"
	} else if hasDownloadRate && !hasDownloadCeil {
		downloadCeil = downloadRate
	} else if hasDownloadRate && !hasDownloadCeil {
		downloadRate = downloadCeil
	}

	if !hasLatencyDelay {
		latencyDelay = "0ms"
	}

	return downloadRate, downloadCeil, uploadRate, uploadCeil, latencyDelay, latencyVariation, latencyCorrelation, lossProbability, lossCorrelation, packetDuplication, packetCorruption, packetReordering
}
