resource "google_project_iam_member" "app_sa_logging_iam" {
  project = var.PROJECT_ID
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.app_sa.email}"
}

