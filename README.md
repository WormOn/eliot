
![Eliot](docs/eliot-logo-medium.png) [![Gitter chat](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/eliot-run)

> This is early alpha version! There's not all features yet implemented, not heavily tested with different devices and code might get large breaking changes until the first release.

Eliot is a open source system for managing containerized applications on top of the IoT device with an emphasis to usability, simplicity, security and stability. Eliot gives simplified app delivery, isolation and additional security to traditional installations.

[![asciicast](https://asciinema.org/a/vZcVZKEfAosSSrhWrJbmIqAd9.png)](https://asciinema.org/a/vZcVZKEfAosSSrhWrJbmIqAd9?autoplay=1&speed=2&t=4)

Docker and Kubernetes have inspired heavily and if you're familiar with those, you find really easy to get started with Eliot.

<sub>Built with ❤︎ by [Erno Aapa](https://github.com/ernoaapa) and [contributors](https://github.com/ernoaapa/eliot/contributors)</sub>

## Usage

- [Documentation](http://docs.eliot.run)
- [Binary releases](https://github.com/ernoaapa/eliot/releases)
- [Docker releases](https://hub.docker.com/r/ernoaapa/eliotd/tags)

Eliot is based on top of the [containerd](https://github.com/containerd/containerd) to provide simple, _Kubernetes like_ API for managing containers.

Eliot is built from following components
- `eli` - Command line tool for managing the device
- `eliotd` - Daemon for the device to manage containers

### Features
- Manage running containers in the device
- Attach to container process remotely for debugging
- Fast _develop-in-device_ development start

[Let us know](https://github.com/ernoaapa/eliot/issues/new) what would be the next awesome feature :)

## Getting started
[See the documentation how to get started](http://docs.eliot.run/getting_started.html)