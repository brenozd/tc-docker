# Traffic Control Docker

[![Docker pulls](https://img.shields.io/docker/pulls/brenozd/tc-docker.svg?label=docker+pulls)](https://hub.docker.com/r/brenozd/tc-docker)
[![Docker stars](https://img.shields.io/docker/stars/brenozd/tc-docker.svg?label=docker+stars)](https://hub.docker.com/r/brenozd/tc-docker)

**Traffic Control Docker** - Network limits for individual docker containers

This repo was originally a fork from [CodyGuo/tc-docker](https://github.com/CodyGuo/tc-docker)

## Running

First run Traffic Control Docker daemon in Docker. The container needs `privileged` capability and the `host` network mode to manage network interfaces on the host system, `/var/run/docker.sock` and `/var/run/docker/netns` volume allows to observe Docker events and query container details.

```bash
docker run -d \
        --name tc-docker \
        --network host \
        --privileged \
        --restart always \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v /var/run/docker/netns:/var/run/docker/netns:shared \
        brenozd/tc-docker
```

> You can also pass 
> * `DOCKER_HOST` and `DOCKER_API_VERSION` environment variables, which default to `unix:///var/run/docker.sock` and `1.40`.
> * `TZ` which defaults to America/Sao_Paulo 

This repository contains `docker-compose.yml` file in root directory, you can use it instead of manually running `docker run` command. Newest version of image will be pulled automatically and the container will run in daemon mode.

```bash
git clone https://github.com/brenozd/tc-docker.git
cd tc-docker
docker-compose up -d
```

## Usage

After the daemon is up it scans all running containers and starts listening for `container:start` events triggered by Docker Engine. When a new container is up and contains `org.label-schema.tc.enabled` label set to `1`, Traffic Control Docker starts applying network traffic rules according to the rest of the labels from `org.label-schema.tc` namespace it finds.

### Recognized Labels
Traffic Control Docker recognizes the following labels:

* `org.label-schema.tc.enabled` - When set to `1` the container network rules will be set automatically, if any other value or if the label is not specified the container will be ignored
* `org.label-schema.tc.upload` - Bandwidth limit for the container upload (egress traffic)
  * `rate` - The maximum rate at which egress traffic will be sent. 
    * Defaults to **10000mbps**
    * Accepts a floating point number, followed by a unit, or a percentage value of the device's speed (e.g. 70.5%). 
    * Following units are recognized: `bit`, `kbit`, `mbit`, `gbit`, `tbit`, `bps`, `kbps`, `mbps`, `gbps`, `tbps`
  * `ceil` - The maximum rate at which egress traffic will be sent if the system has spare bandwidth.
    * Defaults to **rate**  
    * Accepts a floating point number, followed by a unit, or a percentage value of the device's speed (e.g. 70.5%). 
    * Following units are recognized: `bit`, `kbit`, `mbit`, `gbit`, `tbit`, `bps`, `kbps`, `mbps`, `gbps`, `tbps`
* `org.label-schema.tc.download` - Bandwidth limit for the container download (ingress traffic)
  * `rate` - Maximum rate at which ingress traffic will be received. 
    * Defaults to **10000mbps**
    * Accepts a floating point number, followed by a unit, or a percentage value of the device's speed (e.g. 70.5%). 
    * Following units are recognized: `bit`, `kbit`, `mbit`, `gbit`, `tbit`, `bps`, `kbps`, `mbps`, `gbps`, `tbps`
  * `ceil` - Maximum rate at which ingress traffic will be received if the system has spare bandwidth.
    * Defaults to **rate**  
    * Accepts a floating point number, followed by a unit, or a percentage value of the device's speed (e.g. 70.5%). 
    * Following units are recognized: `bit`, `kbit`, `mbit`, `gbit`, `tbit`, `bps`, `kbps`, `mbps`, `gbps`, `tbps`
* `org.label-schema.tc.latency` - Delays outgoing packets
  * `delay` - Delay to be applied to packets outgoing the network interface 
    * Accepts a floating point number, followed by a unit. If a bare number is used it's unit defaults to `usecs`
    * Following units are recognized: `s`, `sec`, `secs`, `ms`, `msec`, `msecs`, `us`, `usec`, `usecs`
  * `variation` - The limit for the random value to be added to delay  
    * Accepts a floating point number, followed by a unit. If a bare number is used it's unit defaults to `usecs`
    * Following units are recognized: `s`, `sec`, `secs`, `ms`, `msec`, `msecs`, `us`, `usec`, `usecs`
    > This label is ignore if **delay** is not set 
  * `correlation` - The correlation or distribution to be applied to delay variation based on the last packet
    * Accepts a floating point number followed by **%** or one of the following distributions: `normal`, `uniform` or `pareto`
    > This label is ignore if **variation** is not set 

    > When using distribution add **distribution** before your choice. e.g.  org.label-schema.tc.latency.variation=distribution normal

* `org.label-schema.tc.loss` - Losses of outgoing packets
  * `probability` - Independent loss probability to the packets outgoing from network
    * Accepts a floating point number followed by **%**
  * `correlation` - The correlation or distribution to be applied to probability losses based on the last packet as it follows:
    > This label is ignore if **probability** is not set 

<p align="center"><img src="https://latex.codecogs.com/svg.image?Loss_{prob}=Correlation\times&space;LastPacket_{prob}&space;&plus;&space;(1-Correlation)\times&space;Probability"></p>

* `org.label-schema.tc.packet` - Packet related label
  * `duplication` - Probability that packets will be duplicated
    * Accepts a floating point number followed by **%**
  * `corruption` - Probability that packets will get corrupted
    * Accepts a floating point number followed by **%**
  * `reordering` - Probability that packets will get reordered
    * Accepts a floating point number followed by **%**

> Read the [tc command manual](http://man7.org/linux/man-pages/man8/tc.8.html) to get detailed information about parameter types and possible values.

## Examples
Here are some examples on how to run limited containers using `tc-docker`

### Bandwidth
```bash
docker run --rm -it \
    --name tc-test \
    --label "org.label-schema.tc.enabled=1" \
    --label "org.label-schema.tc.download.rate=25mbit" \
    --label "org.label-schema.tc.upload.rate=25mbit" \
    alpine sh -c " \
    apk add speedtest-cli \
    && speedtest"
```

### Latency and Loss
```bash
docker run --rm -it \
    --name tc-test \
    --label "org.label-schema.tc.enabled=1" \
    --label "org.label-schema.tc.latency.delay=50ms" \
    --label "org.label-schema.tc.latency.variation=10ms" \
    --label "org.label-schema.tc.latency.correlation=distribution pareto" \
    --label "org.label-schema.tc.loss.probability=33%" \
    alpine sh -c " \
    ping -c 10 google.com"
```
