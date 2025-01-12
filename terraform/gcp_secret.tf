resource "google_secret_manager_secret" "app_config" {
  secret_id = "${var.APP_NAME}-config"
  labels = {
    app = var.APP_NAME
  }
  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_iam_member" "secret_accessor_admin" {
  project   = google_secret_manager_secret.app_config.project
  secret_id = google_secret_manager_secret.app_config.secret_id
  role      = "roles/secretmanager.admin"
  member    = "serviceAccount:${google_service_account.app_sa.email}"
}



