resource "google_project_iam_member" "app_sa_cloudtrace_iam" {
  project = var.PROJECT_ID
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${google_service_account.app_sa.email}"
}

