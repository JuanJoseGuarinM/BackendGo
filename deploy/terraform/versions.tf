# PROVIDER: el "plugin" que sabe hablar con una plataforma (AWS, Azure,
# Kubernetes, Docker...). Aquí usamos el de Docker para practicar en local
# sin una cuenta en la nube. La idea es la misma con cualquier provider.
terraform {
  required_version = ">= 1.5"

  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0" # fijar la versión: el mismo código siempre produce lo mismo
    }
  }

  # STATE (estado): Terraform guarda en un archivo qué creó y con qué ID real.
  # Así sabe qué cambiar o borrar la próxima vez.
  # Aquí es local (terraform.tfstate, ignorado en Git). En un equipo real va
  # en un backend REMOTO compartido y con bloqueo, por ejemplo:
  #
  #   backend "s3" {
  #     bucket         = "mi-empresa-terraform-state"
  #     key            = "orders/terraform.tfstate"
  #     dynamodb_table = "terraform-locks"   # evita que dos personas apliquen a la vez
  #   }
}

provider "docker" {
  host = var.docker_host
}
