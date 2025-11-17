terraform {
  required_providers {
    yandex = {
      source = "yandex-cloud/yandex" // глобальный адрес источника провайдера
    }
  }
  required_version = ">= 0.13" // версия, совместимая с провайдером версия Terraform


  backend "s3" {
    endpoints = {
      s3 = "https://storage.yandexcloud.net"
    }
    bucket         = "maxiks-backet"
    key            = "states/terraform.tfstate"
    use_path_style = true
    region         = "us-east-1" 
  }

}

provider "yandex" {
  zone = "ru-central1-a" // зона доступности по-умолчанию, где будут создаваться ресурсы
  cloud_id  = var.cloud_id
  folder_id = var.folder_id
  token     = var.token
}

resource "yandex_vpc_network" "network-1" {
  name = "network1"
}
resource "yandex_vpc_address" "addr" {
  name = "test"
  deletion_protection = "false"
  external_ipv4_address {
    zone_id = "ru-central1-a"
  }
}
resource "yandex_vpc_subnet" "subnet-1" {
  name           = "subnet1"
  zone           = "ru-central1-a"
  network_id     = yandex_vpc_network.network-1.id
  v4_cidr_blocks = ["192.168.10.0/24"]
}

resource "yandex_compute_disk" "boot-disk-1" {
  name     = "boot-disk-1"
  type     = "network-hdd"
  zone     = "ru-central1-a"
  size     = "20"
  image_id = "fd80tpcdvop5e9qcosnq"
}

resource "yandex_compute_instance" "vm-1" {
  name        = "really-cool-vm1"
  platform_id = "standard-v3"
  resources {
    cores         = 2
    memory        = 2
    core_fraction = 20
  }

  boot_disk {
    disk_id = yandex_compute_disk.boot-disk-1.id
  }

  network_interface {
    subnet_id = yandex_vpc_subnet.subnet-1.id
    nat       = true
    nat_ip_address = yandex_vpc_address.addr.external_ipv4_address[0].address
  }
  metadata = {
    user-data = "${file("conf/meta.txt")}"
  }
  
}

output "ip_address" {
    description = "Static IP"
    value       = yandex_compute_instance.vm-1.network_interface[0].nat_ip_address
}
