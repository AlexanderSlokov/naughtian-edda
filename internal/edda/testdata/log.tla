---- MODULE log ----
\* Edda translated this module from log.hcl. Do not edit it.
EXTENDS Naturals, Sequences

VARIABLES log, buffer, acked
vars == <<log, buffer, acked>>

\* log.hcl:1 state
Init ==
    /\ log = <<>>
    /\ buffer = <<>>
    /\ acked = 0

\* log.hcl:7 step "Write"
Write ==
    /\ buffer' = Append(buffer, "msg")
    /\ UNCHANGED <<log, acked>>

\* log.hcl:11 step "Fsync"
Fsync ==
    /\ log' = log \o buffer
    /\ buffer' = <<>>
    /\ UNCHANGED <<acked>>

\* log.hcl:18 step "Acknowledge"
Acknowledge ==
    /\ Len(log) > acked
    /\ acked' = acked + 1
    /\ UNCHANGED <<log, buffer>>

\* log.hcl:23 step "Crash"
Crash ==
    /\ buffer' = <<>>
    /\ UNCHANGED <<log, acked>>

\* log.hcl:27 invariant "an acknowledged record is durable"
AnAcknowledgedRecordIsDurable == acked <= Len(log)

\* log.hcl:31 constraint
Constraint == Len(log) + Len(buffer) <= 3

Next == Write \/ Fsync \/ Acknowledge \/ Crash

Spec == Init /\ [][Next]_vars
====
