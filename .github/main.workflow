workflow "New workflow" {
  resolves = ["build-and-push"]
  on = "push"
}

action "build-and-push" {
  uses = "pangzineng/Github-Action-One-Click-Docker@master"
  secrets = [
    "DOCKER_USERNAME", 
    "DOCKER_PASSWORD"
  ]
}
