resource "google_sql_database" "database" {
  name     = var.APP_NAME
  instance = "services-db-instance"
}

resource "random_password" "password" {
  length           = 16
  special          = false
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

resource "google_sql_user" "user" {
  name     = var.APP_NAME
  instance = "services-db-instance"
  password = random_password.password.result
}

resource "google_secret_manager_secret" "db_secret" {
  secret_id = "${var.APP_NAME}-db-config"

  labels = {
    app = var.APP_NAME
  }

  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_version" "db_secret_version" {
  secret      = google_secret_manager_secret.db_secret.name
  secret_data = "postgresql://${google_sql_user.user.name}:${google_sql_user.user.password}@${var.DB_HOST}/${google_sql_database.database.name}"
}

resource "google_secret_manager_secret_iam_member" "db_secret_member" {
  project   = google_secret_manager_secret.db_secret.project
  secret_id = google_secret_manager_secret.db_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.app_sa.email}"
}

resource "google_secret_manager_secret" "db_go_cn" {
  secret_id = "${var.APP_NAME}-db-go-cn"

  labels = {
    app = var.APP_NAME
  }

  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_version" "db_go_cn_version" {
  secret      = google_secret_manager_secret.db_go_cn.name
  secret_data = "host=${var.DB_HOST} user=${google_sql_user.user.name} password=${google_sql_user.user.password} dbname=${google_sql_database.database.name} sslmode=disable"
}

resource "google_secret_manager_secret_iam_member" "db_go_cn_member" {
  project   = google_secret_manager_secret.db_go_cn.project
  secret_id = google_secret_manager_secret.db_go_cn.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.app_sa.email}"
}
