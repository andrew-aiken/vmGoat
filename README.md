# vmGoat
![GitHub commits since latest release](https://img.shields.io/github/commits-since/andrew-aiken/vmGoat/latest)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/andrew-aiken/vmGoat/total)
![GitHub Repo stars](https://img.shields.io/github/stars/andrew-aiken/vmGoat)


vmGoat is a project to deploy cyber security scenarios on cloud infrastructure.

- [Install](#install)
- [Scenarios](#scenarios)
- [Contributing](/CONTRIBUTING.md)

> [!WARNING]
> Never use production cloud credentials when using this tool!
> 
> This projects attempts to minimize the blast radius of malicious scenarios though containerization and code review. But your encouraged to review all code that you run.

This project will deploy one ore more virtual machines to a cloud provider (AWS) and then run Ansible against it to create a custom challenge.
I created this project because I originally wanted to publish a [TryHackMe](https://tryhackme.com/) room but my original scenario has not been reviewed over a year after submitting it.

This project is **not** for deploying insecure cloud resources (check out [CloudGoat](https://github.com/RhinoSecurityLabs/cloudgoat/) for that), but instead for configuring virtual environments.

## Install

For the scenarios to be deployed you will need [Docker](https://docs.docker.com/engine/install/) installed and have an [AWS profile](https://wellarchitectedlabs.com/common/documentation/aws_credentials/#files) configured (AWS CLI not required).

```bash
# Downloads the deployment binary
curl https://raw.githubusercontent.com/andrew-aiken/vmGoat/refs/heads/main/install.sh | bash

# Setups a file that contains your IP whitelist
./vmGoat config allowlist

# Setup the AWS profile and region
./vmGoat config aws
```

### Running Locally
If you don't want to add the additional overhead of having Docker installed you can run the application locally.

You will need [Ansible](https://docs.ansible.com/) and passlib installed and then add `--local` to commands that would use docker (create, destroy, purge)

```bash
git clone git@github.com:andrew-aiken/vmGoat.git

cd vmGoat
sh ./install.sh

./vmGoat create --local XYZ
```

### Running from Scratch
In addition to the local dependencies you will also need [Golang](https://go.dev/) installed.
By default, the binary attempts to run inside a container built by GitHub CI.
To run it directly on your local machine, be sure to include the `--local` flag.

```bash
git clone git@github.com:andrew-aiken/vmGoat.git

cd vmGoat

go build -o vmGoat cmd/vmGoat/main.go
./vmGoat create --local XYZ
```

### Running Entirely in Docker


```bash
docker volume create vmGoat

docker run --rm -it --entrypoint bash \
    -v vmGoat:/.config/vmGoat/ \
    -v $HOME/.aws:/root/.aws/:ro \
    --workdir /mnt/ \
    -e VMGOAT_LOCAL=true \
    ghcr.io/andrew-aiken/vmgoat:latest
```

Then run all commands like you normally would except run the binary from `/vmGoat`
The settings will get persisted across deployments of the container.

## Scenarios

### [GitOops](scenarios/gitoops/README.md)
Difficulty: 7/10

In this scenario you discover a unprotected version control system, then using the new access discover a misconfiguration in a continuous deployment system that leads to privileged command execution.
