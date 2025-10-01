group "default" {
  targets = ["backend", "worker", "frontend"]
}

# Enable concurrent builds and optimize for cross-compilation
variable "BUILDKIT_SBOM_SCAN_STAGE" {
  default = "false"
}

variable "BUILDKIT_SBOM_SCAN_CONTEXT" {
  default = "false"
}

variable "APP_VERSION" {
  default = "dev"
}

variable "COMMIT_HASH" {
  default = ""
}

variable "BUILD_TIME" {
  default = ""
}


target "backend" {
  context = "."
  dockerfile = "Dockerfile.backend"
  platforms = ["linux/arm64"]
  args = { APP_VERSION = "${APP_VERSION}", COMMIT_HASH = "${COMMIT_HASH}", BUILD_TIME = "${BUILD_TIME}" }
  tags = ["mrwetsnow/quiz-backend:${APP_VERSION}", "mrwetsnow/quiz-backend:latest"]
}

target "worker" {
  context = "."
  dockerfile = "Dockerfile.worker"
  platforms = ["linux/arm64"]
  args = { APP_VERSION = "${APP_VERSION}", COMMIT_HASH = "${COMMIT_HASH}", BUILD_TIME = "${BUILD_TIME}" }
  tags = ["mrwetsnow/quiz-worker:${APP_VERSION}", "mrwetsnow/quiz-worker:latest"]
}

target "frontend" {
  context = "."
  dockerfile = "Dockerfile.frontend"
  platforms = ["linux/arm64"]
  args = { APP_VERSION = "${APP_VERSION}", COMMIT_HASH = "${COMMIT_HASH}", BUILD_TIME = "${BUILD_TIME}" }
  tags = ["mrwetsnow/quiz-frontend:${APP_VERSION}", "mrwetsnow/quiz-frontend:latest"]
}


