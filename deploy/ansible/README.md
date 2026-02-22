# Ansible deployment (Docker images + host-mounted SQLite)

This playbook deploys DDash and supporting services using Docker Compose, with SQLite files stored on host-mounted paths.

## What is host-mounted

- DDash DB directory: `{{ ddash_data_dir }}` (contains `ddash.sqlite`)
- GitHub ingestor DB directory: `{{ github_ingestor_data_dir }}` (contains `githubapp-ingestor.sqlite`)
- Prometheus and Grafana data are also host-mounted.

## Prerequisites on target host

- Docker with Compose plugin installed.
- SSH access for Ansible user (with sudo).

## Setup

```bash
cd deploy/ansible
ansible-galaxy collection install -r requirements.yml
cp group_vars/all.example.yml group_vars/all.yml
cp inventory.example.ini inventory.ini
```

Edit:

- `group_vars/all.yml` (secrets, URLs, image tags)
- `inventory.ini` (target host/user)

## Deploy

```bash
ansible-playbook -i inventory.ini playbook.yml
```

## Notes

- Compose files are placed on host at `/opt/ddash` by default.
- You can override variables with `-e key=value` or host/group vars.
- For pinned releases, set:
  - `ddash_image: ghcr.io/<owner>/ddash-server:<version>`
  - `github_ingestor_image: ghcr.io/<owner>/ddash-githubappingestor:<version>`
