# OUTPUTS: datos útiles que Terraform muestra al terminar (y que otros
# módulos pueden leer).
output "api_url" {
  description = "URL de la API levantada por Terraform"
  value       = "http://localhost:${var.api_port}"
}

output "containers" {
  description = "Contenedores creados"
  value       = [docker_container.api.name, docker_container.catalog.name]
}
