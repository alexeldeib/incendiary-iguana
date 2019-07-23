workflow "New workflow" {
  resolves = ["Conform Action"]
  on = "push"
}

action "Conform Action" {
  uses = "./actions/conform"
}
