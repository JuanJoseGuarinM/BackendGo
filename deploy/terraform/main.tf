# RESOURCES: lo que quieres que EXISTA. Es DECLARATIVO: describes el
# resultado final ("quiero una red y dos contenedores"), no los pasos.
# Terraform compara esto con el state y calcula qué crear, cambiar o borrar.
#
# Ciclo de trabajo:
#   terraform init      → descarga el provider
#   terraform plan      → muestra QUÉ haría, sin tocar nada (revisar siempre)
#   terraform apply     → lo hace y actualiza el state
#   terraform destroy   → borra todo lo que creó

# Red privada: los contenedores se encuentran por nombre (DNS interno),
# igual que los Services de Kubernetes.
resource "docker_network" "orders" {
  name = "orders-tf"
}

# Imágenes ya construidas en local: keep_locally evita que "destroy" las borre.
resource "docker_image" "api" {
  name         = var.api_image
  keep_locally = true
}

resource "docker_image" "catalog" {
  name         = var.catalog_image
  keep_locally = true
}

resource "docker_container" "catalog" {
  name  = "catalog-tf"
  image = docker_image.catalog.image_id
  env   = ["PORT=8080"]

  networks_advanced {
    name = docker_network.orders.name
  }
}

resource "docker_container" "api" {
  name  = "orders-api-tf"
  image = docker_image.api.image_id

  # Referenciar otro recurso crea una DEPENDENCIA: Terraform crea primero el
  # catálogo y luego la API, sin que se lo digamos.
  env = [
    "PORT=8080",
    "CATALOG_URL=http://${docker_container.catalog.name}:8080",
  ]

  ports {
    internal = 8080
    external = var.api_port
  }

  networks_advanced {
    name = docker_network.orders.name
  }
}
