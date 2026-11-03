# VARIABLES: los valores que cambian entre entornos (dev, QA, prod).
# El código es el mismo; cambia el archivo .tfvars.

variable "docker_host" {
  description = "Cómo conectarse a Docker. En Windows nativo: npipe:////./pipe/docker_engine"
  type        = string
  default     = "unix:///var/run/docker.sock"
}

variable "api_image" {
  description = "Imagen de la API (construida con: docker build -t orders-api:dev .)"
  type        = string
  default     = "orders-api:dev"
}

variable "catalog_image" {
  description = "Imagen del catálogo (docker build -t catalog:dev --build-arg TARGET=catalog .)"
  type        = string
  default     = "catalog:dev"
}

variable "api_port" {
  description = "Puerto de tu máquina donde queda expuesta la API"
  type        = number
  default     = 8097

  validation {
    condition     = var.api_port > 1024 && var.api_port < 65536
    error_message = "api_port debe estar entre 1025 y 65535."
  }
}
