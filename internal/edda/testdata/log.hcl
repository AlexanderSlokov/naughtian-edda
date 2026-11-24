state {
  log    = []
  buffer = []
  acked  = 0
}

step "Write" {
  after = { buffer = append(buffer, "msg") }
}

step "Fsync" {
  after = {
    log    = concat(log, buffer)
    buffer = []
  }
}

step "Acknowledge" {
  when  = length(log) > acked
  after = { acked = acked + 1 }
}

step "Crash" {
  after = { buffer = [] }
}

invariant "an acknowledged record is durable" {
  check = acked <= length(log)
}

constraint {
  check = length(log) + length(buffer) <= 3
}
