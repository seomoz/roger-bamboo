![bamboo-logo](https://cloud.githubusercontent.com/assets/37033/4110258/a8cc58bc-31ef-11e4-87c9-dd20bd2468c2.png)

roger-bamboo is a service discovery and distributed load balancing
system for Roger, Moz's ClusterOS. It is based on
[Bamboo](https://github.com/QubitProducts/bamboo). Roger-bamboo adds
several features to Bamboo including, flexible TCP and HTTP port
specification.

## Deployment

First, get the code from github (if necessary). This requires a valid
go environment on the machine.

```bash
go get github.com/seomoz/roger-bamboo
```

Build and run the genconfig tool to ensure that the HAProxy template
renders correctly against the live marathon data.

```bash
cd $GOPATH/src/github.com/seomoz/roger-bamboo/utils
go build genconfig.go
./genconfig --config config.json
```

This will get data from the live marathon instance and render the template. Change the config.json file in the utils directory to control which Marathon instance to read data from and the path to the HAProxy config template.

The recommended way to install roger-bamboo is with the deb or rpm
package and the
[deb package build script](https://github.com/QubitProducts/bamboo/blob/master/builder/build.sh).
Read comments in the script to customize your build distribution
workflow.

In short, [install fpm](https://github.com/jordansissel/fpm) and run the following command from the root of the roger-bamboo directory:

```bash
go build bamboo.go
./builder/build.sh
```

A deb package will be generated in `./builder` directory. You can copy to a server or publish to your own apt repository.

Moreover, there is
- a [Docker build container](builder/build.sh) which will generate the deb package in the volume mounted output directory.
- and a [vagrant vm](Vagrantfile) where you could call docker build and docker run and build an ubuntu 14.04 binary.
```
vagrant up
vagrant ssh
cd /vagrant
sudo docker build -f Dockerfile-deb -t bamboo-build .
sudo docker run -it -v $(pwd)/output:/output -e "_BAMBOO_VERSION=1.0.3" bamboo-build
```

Independently of how you build the deb package, you can copy it to a server or publish to your own apt repository.

The example deb package deploys:

* Upstart job [`bamboo-server`](https://github.com/QubitProducts/bamboo/blob/master/builder/bamboo-server), e.g. upstart assumes `/var/bamboo/production.json` is configured correctly.
* Application directory is under `/opt/bamboo/`
* Configuration and logs is under `/var/bamboo/`
* Log file is rotated automatically

In case you're not using upstart, a template init.d service is provided in [`init.d-bamboo-server`](https://github.com/QubitProducts/bamboo/blob/master/builder/init.d-bamboo-server). Install it with
```
sudo cp builder/init.d-bamboo-server /etc/init.d/bamboo-server
sudo chown root:root /etc/init.d/bamboo-server
sudo chmod 755 /etc/init.d/bamboo-server
sudo update-rc.d "bamboo-server" defaults
```

You can then start the server with ```sudo service bamboo-server start```. Other commands: status, restart, stop

## Specifying ports
For each deployed app, an arbitrary number of TCP
ports and a single HTTP port can be specified. The specification of
the TCP and HTTP ports is controlled by the `TCP_PORTS` and `HTTP_PORT`
environment variables in the Marathon app config.

### TCP Ports

TCP ports are specified using the `TCP_PORTS`
environment variable. The value of `TCP_PORTS` is expected to be a
JSON object/map with each entry within the object having the format
`externalPort: port_specifier`.

`externalPort` is a port number which is the external port on HAProxy which is opened.

`port_specifier` is a string which is either of the for PORTXX or a number.

If `port_description` is a number then its returned as is. If
  `port_description` is for the form PORTXX where XX is a number, then
  `externalPort` is mapped to `Ports[xx]` where `Ports` is the set of
  ports defined in the Marathon config.

#### Example

```Javascript
{
  "container": {
    "type": "DOCKER",
    "docker": {
      "image": "docker-registry-machine:5000/ncat-v1",
      "network": "BRIDGE",
      "portMappings": [
	{
          "containerPort": 7777,
          "hostPort": 0,
          "servicePort": 0,
          "protocol": "tcp"
        },
        {
          "containerPort": 7778,
          "hostPort": 0
        }
      ]
    },
    "ports": [ 0, 0 ]
  },
  "id": "ncat",
  "instances": 2,
  "cpus": 1,
  "mem": 128,
  "uris": [],
  "env": {
      "TCP_PORTS": "{ \"3300\": \"PORT0\", \"3301\": \"PORT1\", \"3303\": \"21222\"}",
	  "HTTP_PORT": "PORT0"
  }
}
```

Note that the backslashes in the specification string of TCP_PORTS is
necessary as the value of `TCP_PORTS` must be a well formed JSON
object.

Here, three TCP and one HTTP port will be defined.

Port 3300 on HAProxy will map to the first port defined in the ports array.

Port 3301 on HAProxy will map to the second port defined in the ports array.

Port 3303 on HAProxy will map to port 21222 on each machine where the ncat task is running.

The url path /ncat will map to the first port defined in the ports array.
