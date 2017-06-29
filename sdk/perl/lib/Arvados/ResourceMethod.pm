# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

package Arvados::ResourceMethod;
use Carp;
use Data::Dumper;

sub new
{
    my $class = shift;
    my $self = {};
    bless ($self, $class);
    return $self->_init(@_);
}

sub _init
{
    my $self = shift;
    $self->{'resourceAccessor'} = shift;
    $self->{'method'} = shift;
    return $self;
}

sub execute
{
    my $self = shift;
    my $method = $self->{'method'};

    my $path = $method->{'path'};

    my %body_params;
    my %given_params = @_;
    my %extra_params = %given_params;
    my %method_params = %{$method->{'parameters'}};
    if ($method->{'request'}->{'properties'}) {
        while (my ($prop_name, $prop_value) =
               each %{$method->{'request'}->{'properties'}}) {
            if (ref($prop_value) eq 'HASH' && $prop_value->{'$ref'}) {
                $method_params{$prop_name} = { 'type' => 'object' };
            }
        }
    }
    while (my ($param_name, $param) = each %method_params) {
        delete $extra_params{$param_name};
        if ($param->{'required'} && !exists $given_params{$param_name}) {
            croak("Required parameter not supplied: $param_name");
        }
        elsif ($param->{'location'} eq 'path') {
            $path =~ s/{\Q$param_name\E}/$given_params{$param_name}/eg;
        }
        elsif (!exists $given_params{$param_name}) {
            ;
        }
        elsif ($param->{'type'} eq 'object') {
            my %param_value;
            my ($p, $v);
            if (exists $param->{'properties'}) {
                while (my ($property_name, $property) =
                       each %{$param->{'properties'}}) {
                    # if the discovery doc specifies object structure,
                    # convert to true/false depending on supplied type
                    if (!exists $given_params{$param_name}->{$property_name}) {
                        ;
                    }
                    elsif (!defined $given_params{$param_name}->{$property_name}) {
                        $param_value{$property_name} = JSON::null;
                    }
                    elsif ($property->{'type'} eq 'boolean') {
                        $param_value{$property_name} = $given_params{$param_name}->{$property_name} ? JSON::true : JSON::false;
                    }
                    else {
                        $param_value{$property_name} = $given_params{$param_name}->{$property_name};
                    }
                }
            }
            else {
                while (my ($property_name, $property) =
                       each %{$given_params{$param_name}}) {
                    if (ref $property eq '' || $property eq undef) {
                        $param_value{$property_name} = $property;
                    }
                    elsif (ref $property eq 'HASH') {
                        $param_value{$property_name} = {};
                        while (my ($k, $v) = each %$property) {
                            $param_value{$property_name}->{$k} = $v;
                        }
                    }
                }
            }
            $body_params{$param_name} = \%param_value;
        } elsif ($param->{'type'} eq 'boolean') {
            $body_params{$param_name} = $given_params{$param_name} ? JSON::true : JSON::false;
        } else {
            $body_params{$param_name} = $given_params{$param_name};
        }
    }
    if (%extra_params) {
        croak("Unsupported parameter(s) passed to API call /$path: \"" . join('", "', keys %extra_params) . '"');
    }
    my $r = $self->{'resourceAccessor'}->{'api'}->new_request;
    my $base_uri = $self->{'resourceAccessor'}->{'api'}->{'discoveryDocument'}->{'baseUrl'};
    $base_uri =~ s:/$::;
    $r->set_uri($base_uri . "/" . $path);
    $r->set_method($method->{'httpMethod'});
    $r->set_auth_token($self->{'resourceAccessor'}->{'api'}->{'authToken'});
    $r->set_query_params(\%body_params) if %body_params;
    $r->process_request();
    my $data, $headers;
    my ($status_number, $status_phrase) = $r->get_status();
    if ($status_number != 200) {
        croak("API call /$path failed: $status_number $status_phrase\n". $r->get_body());
    }
    $data = $r->get_body();
    $headers = $r->get_headers();
    my $result = JSON::decode_json($data);
    if ($method->{'response'}->{'$ref'} =~ /List$/) {
        Arvados::ResourceProxyList->new($result, $self->{'resourceAccessor'});
    } else {
        Arvados::ResourceProxy->new($result, $self->{'resourceAccessor'});
    }
}

1;
