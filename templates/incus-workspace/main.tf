terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = "2.18.0"
    }
    incus = {
      source  = "lxc/incus"
      version = "1.2.0"
    }
  }
}

provider "coder" {}
provider "incus" {}

data "coder_workspace" "me" {}
data "coder_workspace_owner" "me" {}

data "coder_parameter" "cpu" {
  name         = "cpu"
  display_name = "CPU cores"
  description  = "CPU limit for this isolated workspace."
  type         = "number"
  default      = "4"
  mutable      = true
  order        = 1
  option {
    name  = "2 cores"
    value = "2"
  }
  option {
    name  = "4 cores"
    value = "4"
  }
  option {
    name  = "8 cores"
    value = "8"
  }
}

data "coder_parameter" "memory" {
  name         = "memory"
  display_name = "Memory (GiB)"
  description  = "Memory limit for this isolated workspace."
  type         = "number"
  default      = "8"
  mutable      = true
  order        = 2
  option {
    name  = "4 GiB"
    value = "4"
  }
  option {
    name  = "8 GiB"
    value = "8"
  }
  option {
    name  = "16 GiB"
    value = "16"
  }
}

data "coder_parameter" "service_port" {
  name         = "service_port"
  display_name = "Web service port"
  description  = "Port exposed through the HTTPS workspace application."
  type         = "number"
  default      = "8000"
  mutable      = true
  order        = 3
  validation {
    min = 1024
    max = 65535
  }
}

data "coder_parameter" "service_visibility" {
  name         = "service_visibility"
  display_name = "Web service visibility"
  description  = "owner is private; authenticated allows platform users; public allows anonymous HTTPS access."
  type         = "string"
  default      = "owner"
  mutable      = true
  order        = 4
  option {
    name  = "Private to owner"
    value = "owner"
  }
  option {
    name  = "Authenticated platform users"
    value = "authenticated"
  }
  option {
    name  = "Public anonymous access"
    value = "public"
  }
}

locals {
  instance_name = substr("tc-${data.coder_workspace_owner.me.name}-${data.coder_workspace.me.name}", 0, 63)
  home_name     = substr("home-${data.coder_workspace_owner.me.id}-${data.coder_workspace.me.id}", 0, 63)
}

resource "coder_agent" "main" {
  arch = "amd64"
  os   = "linux"

  display_apps {
    web_terminal           = true
    ssh_helper             = true
    port_forwarding_helper = true
  }

  metadata {
    display_name = "CPU Usage"
    key          = "cpu_usage"
    script       = "coder stat cpu"
    interval     = 10
    timeout      = 1
    order        = 1
  }

  metadata {
    display_name = "RAM Usage"
    key          = "ram_usage"
    script       = "coder stat mem"
    interval     = 10
    timeout      = 1
    order        = 2
  }

  resources_monitoring {
    memory {
      enabled   = true
      threshold = 90
    }
    volume {
      path      = "/home/coder"
      enabled   = true
      threshold = 90
    }
  }
}

resource "coder_app" "service" {
  agent_id     = coder_agent.main.id
  slug         = "service"
  display_name = "Workspace Service"
  url          = "http://127.0.0.1:${data.coder_parameter.service_port.value}"
  subdomain    = true
  share        = data.coder_parameter.service_visibility.value

  healthcheck {
    url       = "http://127.0.0.1:${data.coder_parameter.service_port.value}"
    interval  = 5
    threshold = 12
  }
}

resource "incus_storage_volume" "home" {
  name         = local.home_name
  pool         = "tcompute"
  project      = "user-955"
  content_type = "filesystem"
  description  = "Persistent home for ${data.coder_workspace_owner.me.name}/${data.coder_workspace.me.name}"
  config = {
    size = "50GiB"
  }
}

resource "incus_instance" "workspace" {
  count = data.coder_workspace.me.start_count

  name      = local.instance_name
  project   = "user-955"
  image     = "ec826a760fb7be23086d5e5c032dab471d147884b3342c75f352c19490bddc10"
  type      = "container"
  profiles  = ["default"]
  ephemeral = false
  running   = true

  config = {
    "boot.autostart"           = "false"
    "limits.cpu"               = data.coder_parameter.cpu.value
    "limits.memory"            = "${data.coder_parameter.memory.value}GiB"
    "limits.processes"         = "2048"
    "security.idmap.isolated"  = "true"
    "security.nesting"         = "true"
    "security.privileged"      = "false"
    "user.tcompute.owner_id"   = data.coder_workspace_owner.me.id
    "user.tcompute.owner_name" = data.coder_workspace_owner.me.name
    "user.tcompute.workspace"  = data.coder_workspace.me.name
  }

  device {
    name = "home"
    type = "disk"
    properties = {
      path   = "/home/coder"
      source = incus_storage_volume.home.name
      pool   = incus_storage_volume.home.pool
    }
  }

  wait_for {
    type = "ipv4"
    nic  = "eth0"
  }

  file {
    target_path = "/etc/tcompute-agent-token"
    content     = coder_agent.main.token
    uid         = 0
    gid         = 0
    mode        = "0600"
  }

  file {
    target_path = "/etc/systemd/system/tcompute-user-service.service"
    uid         = 0
    gid         = 0
    mode        = "0644"
    content     = <<-UNIT
      [Unit]
      Description=Tashan Compute persistent user service
      After=network-online.target
      Wants=network-online.target
      ConditionPathIsExecutable=/home/coder/.tcompute/service

      [Service]
      Type=simple
      User=root
      Group=root
      WorkingDirectory=/home/coder
      ExecStart=/home/coder/.tcompute/service
      Restart=always
      RestartSec=5

      [Install]
      WantedBy=multi-user.target
    UNIT
  }

  file {
    target_path = "/etc/systemd/system/tcompute-coder-agent.service"
    uid         = 0
    gid         = 0
    mode        = "0644"
    content     = <<-UNIT
      [Unit]
      Description=Tashan Compute Coder workspace agent
      After=network-online.target
      Wants=network-online.target

      [Service]
      Type=simple
      User=root
      Group=root
      WorkingDirectory=/home/coder
      Environment=CODER_AGENT_TOKEN_FILE=/etc/tcompute-agent-token
      Environment=CODER_AGENT_URL=https://compute.tashan.chat
      Environment=CODER_AGENT_AUTH=token
      ExecStart=/usr/local/bin/coder agent
      Restart=always
      RestartSec=5

      [Install]
      WantedBy=multi-user.target
    UNIT
  }

  exec = {
    "10-home-owner" = {
      command = ["chown", "coder:coder", "/home/coder"]
      trigger = "once"
    }
    "20-systemd-reload" = {
      command = ["systemctl", "daemon-reload"]
      trigger = "once"
    }
    "30-agent-enable" = {
      command = ["systemctl", "enable", "--now", "tcompute-coder-agent.service"]
      trigger = "once"
    }
    "40-user-service-enable" = {
      command = ["systemctl", "enable", "--now", "tcompute-user-service.service"]
      trigger = "once"
    }
    "50-boundary-check" = {
      command = ["/usr/local/sbin/tcompute-boundary-check"]
      trigger = "once"
    }
  }
}

resource "coder_metadata" "workspace" {
  count       = data.coder_workspace.me.start_count
  resource_id = incus_instance.workspace[0].name

  item {
    key   = "instance"
    value = incus_instance.workspace[0].name
  }
  item {
    key   = "runtime"
    value = "Incus unprivileged container"
  }
  item {
    key   = "persistent_home"
    value = "50 GiB"
  }
}
