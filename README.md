[![License](https://img.shields.io/github/license/yorick1989/cod_masterserver?style=for-the-badge&color=green "License")](https://github.com/yorick1989/cod_masterserver/blob/main/LICENSE)[![Amount of contributers](https://img.shields.io/github/contributors/yorick1989/cod_masterserver?style=for-the-badge&color=blue "Amount of contributers")](https://github.com/yorick1989/cod_masterserver/graphs/contributors)[![Amount of downloads](https://img.shields.io/github/downloads/yorick1989/cod_masterserver/total?style=for-the-badge&color=blue "Amount of downloads")](https://github.com/yorick1989/cod_masterserver)[![Forks](https://img.shields.io/github/forks/yorick1989/cod_masterserver?style=for-the-badge&color=blue "Forks")](https://github.com/yorick1989/cod_masterserver/forks)  \
[![Last commit](https://img.shields.io/github/last-commit/yorick1989/cod_masterserver?style=for-the-badge "Last commit")](https://github.com/yorick1989/cod_masterserver/commits/main)[![Latest tag](https://img.shields.io/github/v/tag/yorick1989/cod_masterserver?style=for-the-badge "Latest Tag")](https://github.com/yorick1989/cod_masterserver/tags)[![Latest release](https://img.shields.io/github/v/release/yorick1989/cod_masterserver?style=for-the-badge "Latest release")](https://github.com/yorick1989/cod_masterserver/releases/latest)[![State of the latest release build](https://img.shields.io/github/actions/workflow/status/yorick1989/cod_masterserver/release.yml?style=for-the-badge "State of the latest release build")](https://github.com/yorick1989/cod_masterserver/actions/workflows/release.yml) \
[![Commit activity](https://img.shields.io/github/commit-activity/m/yorick1989/cod_masterserver?style=for-the-badge "Commit activity")](https://github.com/yorick1989/cod_masterserver/graphs/commit-activity)[![Open issues](https://img.shields.io/github/issues/yorick1989/cod_masterserver?style=for-the-badge "Open issues")](https://github.com/yorick1989/cod_masterserver/issues)[![Open Pull Requests](https://img.shields.io/github/issues-pr/yorick1989/cod_masterserver?style=for-the-badge "Open Pull Requests")](https://github.com/yorick1989/cod_masterserver/pulls)

# Call of Duty (2) master server and gameserver browser

## Introduction
I started this project to get more familiar with Golang, (Call of Duty) master servers, how to fetch gameservers and show them in a web interface.

The project was picked up by someone else that didn't go further in developing it in Golang and therefor this project became obsolete.

I've learned a lot; how the Call of Duty master protocol works and did get more falimiar with Golang. I've used serval Github repositories during my learning experience and therefor I published this code so others can maybe learn something from it too. It's lacking testing and it's far from perfect and best practice, but maybe it can help someone in some way too.

### What does the application
The application runs a Call of Duty (2) master server, a web interface that shows all the gameservers that are collected from all the official cod master servers and it also has the option to run an authentication server for Call of Duty gameservers.

### The state of this project
This code is not developed anymore as it is not actively used anymore.

In case you have security or other important/interesting improvements, please create a pull request. I'm always open to learn from others.

## Installation
You can use this application by compiling it yourself or use the container image instead.

### Compilation
You can compile the application by downloading this repository and run the `make all` command.

### Container image
You can download the container image and run the server with, for example, `podman`:
```bash
podman run -p 8080:8080 -d --name cod_masterserver ghcr.io/yorick1989/cod_masterserver
```

## Automatically update and start with OS
In case you're using `podman`, make sure you created the `codmaster` user on your server and that it is allowed to run containers using `podman`.

Install the `systemd` unit file, using the follwing command (adjust accordingly where needed):
```bash
cat << 'EOF' > /etc/systemd/system/cod-masterserver.service && systemctl daemon-reload && systemctl --now enable cod-masterserver.service
[Unit]
Description=COD master server
After=network-online.target
     
[Service]
Type=exec
User=codmaster
Restart=always
ExecStart=/usr/bin/podman run \
    -ti --rm --replace \
    -p 8080:8080/tcp \
    --pull newer \
    --name cod_masterserver \
    ghcr.io/yorick1989/cod_masterserver:latest
ExecStop=/usr/bin/podman stop cod_masterserver
ExecStopPost=/usr/bin/podman system prune \
    --filter "label=org.opencontainers.image.title=Call of Duty master server" \
    --force
     
[Install]
WantedBy=multi-user.target
EOF 
``` 

### Available options for the application
To view the list of options using the help argument:
```bash
podman run ghcr.io/yorick1989/cod_masterserver -h
Usage of /master:
  -codauth
        Enable the Cod auth server
  -codauthip string
        Cod auth server ip (default "0.0.0.0")
  -codauthport int
        Cod auth server port (default 20700)
  -codmasterip string
        Cod master server ip (default "0.0.0.0")
  -codmasterport int
        Cod master server port (default 20710)
  -gscheck duration
        Amount of seconds between the checks of the gameservers. (default 30s)
  -querytimeout duration
        UDP Query timeout (for both COD master and COD update servers) (default 30s)
  -webip string
        The ip where the webserver listens on (default "0.0.0.0")
  -webport int
        The port where the webserver listens on (default 8080)
```

### Example
An example of the web interface of this application can be found at [https://yorick.gruijthuijzen.nl/codmaster](https://yorick.gruijthuijzen.nl/codmaster).

## Contact
[![My Discord (YOURUSERID)](https://img.shields.io/badge/My-Discord-%235865F2.svg)](https://discord.com/users/370120292665917443)
