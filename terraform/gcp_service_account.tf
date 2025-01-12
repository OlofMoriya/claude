resource "google_service_account" "app_sa" {
  project      = var.PROJECT_ID
  account_id   = var.APP_NAME
  display_name = "${var.APP_NAME} Service Account"
}

resource "google_service_account_iam_member" "app_sa_k8s" {
  service_account_id = google_service_account.app_sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.PROJECT_ID}.svc.id.goog[${var.NS}/${var.APP_NAME}]"
}

