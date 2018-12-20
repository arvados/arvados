# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# cpan -I -T install Graph::Easy
# (eval `perl -I ~/perl5/lib/perl5 -Mlocal::lib`; cpan -T install Graph::Easy)
# graph-easy --as=svg < readme_states.txt

[Nonexistent] - appears in cloud list -> [Unknown]
[Nonexistent] - create() returns ID -> [Booting]
[Unknown] - create() returns ID -> [Booting]
[Unknown] - boot timeout -> [Shutdown]
[Booting] - boot+run probes succeed -> [Idle]
[Idle] - idle timeout -> [Shutdown]
[Idle] - probe timeout -> [Shutdown]
[Idle] - want=drain -> [Shutdown]
[Idle] - container starts -> [Running]
[Running] - container ends -> [Idle]
[Running] - container ends, want=drain -> [Shutdown]
[Shutdown] - instance disappears from cloud -> [Gone]

# Layouter fails if we add these
#[Hold] - want=run -> [Booting]
#[Hold] - want=drain -> [Shutdown]
#[Running] - container ends, want=hold -> [Hold]
#[Unknown] - want=hold -> [Hold]
#[Booting] - want=hold -> [Hold]
#[Idle] - want=hold -> [Hold]

# Not worth saying?
#[Booting] - boot probe succeeds, run probe fails -> [Booting]
