swarm-service-restart
=====================

Some applications require periodic restarts to keep things running smoothly, but

Rather than using set of shell scripts or cron jobs, this tool allows you to attach
a label to services (`edu.wm.restartService.schedule`) that contains a [crontab
like scheduling syntax](https://en.wikipedia.org/wiki/Cron#Overview).

The cron library used is a modified of robfig's
[cron.v2 library](https://godoc.org/gopkg.in/robfig/cron.v2). The [cron
syntax](https://godoc.org/gopkg.in/robfig/cron.v2#hdr-CRON_Expression_Format)
is the same.

At this point I'd consider everything not quite alpha stability.

## Requirements

The swarm-service-restart tool has to run on a manager node (or at least have
access to a manager node's API, possibly using something like [mikesir87
describes](https://blog.mikesir87.io/2018/07/letting-traefik-run-on-worker-nodes/)).

The binary can be run directly on the node, or can be run as its own
service (recommended).

## Example

A docker compose file might look something like
```yaml
version: "3.7"

services:
  service-restarter:
    image: pcfens/swarm-service-restart
    deploy:
      placement:
        constraints:
          - node.role == manager
    volumes:
      - type: bind
        source: /var/run/docker.sock
        target: /var/run/docker.sock
        read_only: false
```

A service that will be restarted at 2am everyday would look something like
```yaml
version: "3.7"

services:
  nginx:
    image: nginx
    deploy:
      labels:
        edu.wm.restartService.schedule: "0 2 * * *"
```

The files `docker-compose.yaml` and `test-nginx.yaml` are both used during
development.

## Future Plans

* I'd like to build something like this for Kubernetes, possibly in the same
tool.
* As the code shows, I'm not all that familiar with Go, so there are numerous
opportunities to clean things up.
