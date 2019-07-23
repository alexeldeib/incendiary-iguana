workflow "New workflow" {
  resolves = ["build-and-push"]
  on = "push"
}

action "test" {
  uses = "./actions/test"
  secrets = [
    "DOCKER_USERNAME", 
    "DOCKER_PASSWORD"
  ]
}

action "build-and-push" {
  needs = "test"
  uses = "pangzineng/Github-Action-One-Click-Docker@master"
  secrets = [
    "DOCKER_USERNAME", 
    "DOCKER_PASSWORD"
  ]
}
