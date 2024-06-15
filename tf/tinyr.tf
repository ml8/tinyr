variable "project_id" {
  description = "project id"
}

variable "region" {
  description = "compute region"
}

variable "zone" {
  description = "compute zone"
}

variable "gke_vm_type" {
  description = "vm type for nodepool"
}

variable "gke_num_nodes" {
  description = "number of nodes in nodepool"
}

variable "gke_min_num_nodes" {
  description = "minimum number of nodes in nodepool"
}

variable "gke_max_num_nodes" {
  description = "maximum number of nodes in nodepool"
}

variable "db_disk_size" {
  description = "size of database disk in GB"
}

variable "db_user_password" {
  description = "password for db user"
}

locals {
  default_network = "https://www.googleapis.com/compute/v1/projects/${var.project_id}/global/networks/default"
}

data "google_client_config" "current" {}

resource "google_container_cluster" "primary" {
  name     = "${var.project_id}-gke"
  project = var.project_id
  location = var.zone

  deletion_protection = false

  remove_default_node_pool = true
  initial_node_count = 1

  networking_mode = "VPC_NATIVE"
  ip_allocation_policy {}
}

resource "google_container_node_pool" "primary_nodes" {
  project = var.project_id

  name    = "${google_container_cluster.primary.name}-node-pool"
  cluster = google_container_cluster.primary.name

  location = var.zone

  node_count = var.gke_num_nodes
  autoscaling {
    total_min_node_count = var.gke_min_num_nodes
    total_max_node_count = var.gke_max_num_nodes
  }


  node_config {
    oauth_scopes = [
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/cloud-platform",
      "https://www.googleapis.com/auth/service.management.readonly",
      "https://www.googleapis.com/auth/servicecontrol"
    ]

    labels = {
      env = var.project_id
    }

    preemptible  = true
    machine_type = var.gke_vm_type
    tags         = ["gke-node", "${var.project_id}-gke"]
    metadata = {
      disable-legacy-endpoints = "true"
    }
  }

  depends_on = [
    google_container_cluster.primary
  ]
}

resource "google_compute_global_address" "private_db_ip" {
  project = var.project_id

  name = "sql-static-ip"
  purpose = "VPC_PEERING"
  address_type = "INTERNAL"
  prefix_length = 16
  network = local.default_network
}

resource "google_service_networking_connection" "default" {
  network = local.default_network
  service = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_db_ip.name]
}

resource "google_sql_database_instance" "instance" {
  project = var.project_id

  name = "tinyr-mysql"
  region = var.region

  database_version = "MYSQL_8_0"

  settings {
    tier = "db-f1-micro"
    disk_size = var.db_disk_size
    ip_configuration {
      ipv4_enabled = true
      private_network = local.default_network
      enable_private_path_for_google_cloud_services = true
    }
  }

  deletion_protection = false

  depends_on = [
    google_service_networking_connection.default,
    google_container_node_pool.primary_nodes
  ]
}

resource "google_sql_user" "sql_user" {
  project = var.project_id 

  instance = google_sql_database_instance.instance.name

  name = "tinyr-user"
  password = var.db_user_password

  depends_on = [
    google_sql_database_instance.instance
  ]
}

resource "google_sql_database" "db" {
  project = var.project_id

  name = "tinyr"
  instance = google_sql_database_instance.instance.name
}
