# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

package Arvados::ResourceProxy;

sub new
{
    my $class = shift;
    my $self = shift;
    $self->{'resourceAccessor'} = shift;
    bless ($self, $class);
    $self;
}

sub save
{
    my $self = shift;
    $response = $self->{'resourceAccessor'}->{'update'}->execute('uuid' => $self->{'uuid'}, $self->resource_parameter_name() => $self);
    foreach my $param (keys %$self) {
        if (exists $response->{$param}) {
            $self->{$param} = $response->{$param};
        }
    }
    $self;
}

sub update_attributes
{
    my $self = shift;
    my %updates = @_;
    $response = $self->{'resourceAccessor'}->{'update'}->execute('uuid' => $self->{'uuid'}, $self->resource_parameter_name() => \%updates);
    foreach my $param (keys %updates) {
        if (exists $response->{$param}) {
            $self->{$param} = $response->{$param};
        }
    }
    $self;
}

sub reload
{
    my $self = shift;
    $response = $self->{'resourceAccessor'}->{'get'}->execute('uuid' => $self->{'uuid'});
    foreach my $param (keys %$self) {
        if (exists $response->{$param}) {
            $self->{$param} = $response->{$param};
        }
    }
    $self;
}

sub resource_parameter_name
{
    my $self = shift;
    my $pname = $self->{'resourceAccessor'}->{'resourcesName'};
    $pname =~ s/s$//;           # XXX not a very good singularize()
    $pname;
}

1;
