resource "cloudflare_worker_cron_trigger" "example" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "my-worker"
  cron        = "0 0 * * *"
}

resource "cloudflare_worker_cron_trigger" "hourly" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "hourly-worker"
  cron        = "0 * * * *"
}

resource "cloudflare_workers_cron_trigger" "already_plural" {
  account_id  = "f037e56e89293a057740de681ac9abbe"
  script_name = "already-correct"
  cron        = "*/5 * * * *"
}
