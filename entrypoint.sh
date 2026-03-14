#!/bin/sh
# Setup cgroup v2 hierarchy for lime.
#
# Docker containers start with all processes in the cgroup namespace root.
# cgroup v2's "no internal processes" rule prevents enabling controllers in
# cgroup.subtree_control while processes live there. We must move them out
# first, then enable controllers, then create the lime subtree.

CGROUP_ROOT=/sys/fs/cgroup
LIME_CGROUP="${CGROUP_ROOT}/lime"
PREINIT="${CGROUP_ROOT}/lime.preinit"

# Create a transient holding cgroup and move all root-cgroup processes there.
mkdir -p "$PREINIT"
for pid in $(cat "$CGROUP_ROOT/cgroup.procs" 2>/dev/null); do
    echo "$pid" > "$PREINIT/cgroup.procs" 2>/dev/null || true
done

# Root cgroup is now empty — enable the required controllers.
echo "+cpu +cpuset +memory +pids +io" > "$CGROUP_ROOT/cgroup.subtree_control"

# Create the lime subtree and propagate controllers into it.
mkdir -p "$LIME_CGROUP"
echo "+cpu +cpuset +memory +pids +io" > "$LIME_CGROUP/cgroup.subtree_control"
# Move ourselves into the lime cgroup so per-submission child cgroups
# created under it inherit the enabled controllers.
echo $$ > "$LIME_CGROUP/cgroup.procs" 2>/dev/null || true

export LIME_CGROUP_ROOT="$LIME_CGROUP"
exec /usr/local/bin/worker "$@"
