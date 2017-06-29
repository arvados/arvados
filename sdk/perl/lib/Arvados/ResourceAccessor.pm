# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

package Arvados::ResourceAccessor;
use Carp;
use Data::Dumper;

sub new
{
    my $class = shift;
    my $self = {};
    bless ($self, $class);

    $self->{'api'} = shift;
    $self->{'resourcesName'} = shift;
    $self->{'methods'} = $self->{'api'}->{'discoveryDocument'}->{'resources'}->{$self->{'resourcesName'}}->{'methods'};
    my $method_name, $method;
    while (($method_name, $method) = each %{$self->{'methods'}}) {
        $self->{$method_name} = Arvados::ResourceMethod->new($self, $method);
    }
    $self;
}

1;
