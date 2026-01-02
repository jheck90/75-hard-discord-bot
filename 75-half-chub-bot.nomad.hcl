job "75-half-chub" {
  datacenters = ["dc1"]
  type        = "service"

  constraint {
    attribute = node.unique.name
    value     = "beef-server"
  }

  group "75-half-chub" {
    count = 1

    task "75-half-chub" {
      driver = "docker"
      config {
        image = "jheck90/75-half-chub-bot:v1.0.0"

      }
      template {
          destination = "${NOMAD_SECRETS_DIR}/env.txt"
          env         = true
          data        = <<EOT
          {{- with nomadVar "secret/creds/75-half-chub-bot@default" -}}
          {{- range $k, $v := . }}
          {{ $k | toUpper }}={{ $v }}
          {{- end }}
          {{- end }}
        EOT
        }
      env {
          DB_HOST="192.168.86.3"
          DB_USER="hard75" # Optional, defaults to postgres 
          DB_PASSWORD="hard75" # Required if DB_HOST is set 
          DB_NAME="hard75" # Optional, defaults to hard75 
          DB_SSLMODE="disable"
          # DEV_MODE="dev"
          # LOG_LEVEL="INFO"
        }
      service {
        provider = "nomad"
      }

      resources {
        cpu    = 500
        memory = 300
      }
    }
  }
}