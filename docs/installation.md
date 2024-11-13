# Installation

## Getting started

### Install

#### MacOS

```sh
brew tap taskctl/taskctl
brew install taskctl
```

#### Linux

```sh
sudo wget https://github.com/Ensono/taskctl/releases/latest/download/taskctl_linux_amd64 -O /usr/local/bin/taskctl
sudo chmod +x /usr/local/bin/taskctl
```

#### Ubuntu Linux

```sh
sudo snap install --classic taskctl
```

#### deb/rpm:

Download the .deb or .rpm from the [releases](https://github.com/Ensono/taskctl/releases) page and install with `dpkg -i` 
and `rpm -i` respectively.

#### Windows

```sh
scoop bucket add taskctl https://github.com/taskctl/scoop-taskctl.git
scoop install taskctl
```

#### Installation script

```sh
curl -sL https://raw.githubusercontent.com/taskctl/taskctl/master/install.sh | sh
```

#### From sources

```sh
git clone https://github.com/Ensono/taskctl
cd taskctl
go build -o taskctl .
```

#### Docker images

Docker images available on [Docker hub](https://hub.docker.com/repository/docker/taskctl/taskctl)

### Usage

- `taskctl` - run interactive task prompt
- `taskctl pipeline1` - run single pipeline
- `taskctl task1` - run single task
- `taskctl pipeline1 task1` - run one or more pipelines and/or tasks
- `taskctl watch watcher1 watcher2` - start one or more watchers