workflow "New workflow" {
  resolves = ["Conform Action"]
  on = "push"
}

action "Conform Action" {
  uses = "talos-systems/conform@v0.1.0-alpha.15"
}
