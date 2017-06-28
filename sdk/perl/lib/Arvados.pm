# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

=head1 NAME

Arvados -- client library for Arvados services

=head1 SYNOPSIS

  use Arvados;
  $arv = Arvados->new(apiHost => 'arvados.local');

  my $instances = $arv->{'pipeline_instances'}->{'list'}->execute();
  print "UUID is ", $instances->{'items'}->[0]->{'uuid'}, "\n";

  $uuid = 'eiv0u-arx5y-2c5ovx43zw90gvh';
  $instance = $arv->{'pipeline_instances'}->{'get'}->execute('uuid' => $uuid);
  print "ETag is ", $instance->{'etag'}, "\n";

  $instance->{'active'} = 1;
  $instance->{'name'} = '';
  $instance->save();
  print "ETag is ", $instance->{'etag'}, "\n";

=head1 METHODS

=head2 new()

 my $whc = Arvados->new( %OPTIONS );

Set up a client and retrieve the schema from the server.

=head3 Options

=over

=item apiHost

Hostname of API discovery service. Default: C<ARVADOS_API_HOST>
environment variable, or C<arvados>

=item apiProtocolScheme

Protocol scheme. Default: C<ARVADOS_API_PROTOCOL_SCHEME> environment
variable, or C<https>

=item authToken

Authorization token. Default: C<ARVADOS_API_TOKEN> environment variable

=item apiService

Default C<arvados>

=item apiVersion

Default C<v1>

=back

=cut

package Arvados;

use Net::SSL (); # From Crypt-SSLeay
BEGIN {
  $Net::HTTPS::SSL_SOCKET_CLASS = "Net::SSL"; # Force use of Net::SSL
}

use JSON;
use Carp;
use Arvados::ResourceAccessor;
use Arvados::ResourceMethod;
use Arvados::ResourceProxy;
use Arvados::ResourceProxyList;
use Arvados::Request;
use Data::Dumper;

$Arvados::VERSION = 0.1;

sub new
{
    my $class = shift;
    my %self = @_;
    my $self = \%self;
    bless ($self, $class);
    return $self->build(@_);
}

sub build
{
    my $self = shift;

    $config = load_config_file("$ENV{HOME}/.config/arvados/settings.conf");

    $self->{'authToken'} ||=
	$ENV{ARVADOS_API_TOKEN} || $config->{ARVADOS_API_TOKEN};

    $self->{'apiHost'} ||=
	$ENV{ARVADOS_API_HOST} || $config->{ARVADOS_API_HOST};

    $self->{'noVerifyHostname'} ||=
	$ENV{ARVADOS_API_HOST_INSECURE};

    $self->{'apiProtocolScheme'} ||=
	$ENV{ARVADOS_API_PROTOCOL_SCHEME} ||
	$config->{ARVADOS_API_PROTOCOL_SCHEME};

    $self->{'ua'} = new Arvados::Request;

    my $host = $self->{'apiHost'} || 'arvados';
    my $service = $self->{'apiService'} || 'arvados';
    my $version = $self->{'apiVersion'} || 'v1';
    my $scheme = $self->{'apiProtocolScheme'} || 'https';
    my $uri = "$scheme://$host/discovery/v1/apis/$service/$version/rest";
    my $r = $self->new_request;
    $r->set_uri($uri);
    $r->set_method("GET");
    $r->process_request();
    my $data, $headers;
    my ($status_number, $status_phrase) = $r->get_status();
    $data = $r->get_body() if $status_number == 200;
    $headers = $r->get_headers();
    if ($data) {
        my $doc = $self->{'discoveryDocument'} = JSON::decode_json($data);
        print STDERR Dumper $doc if $ENV{'DEBUG_ARVADOS_API_DISCOVERY'};
        my $k, $v;
        while (($k, $v) = each %{$doc->{'resources'}}) {
            $self->{$k} = Arvados::ResourceAccessor->new($self, $k);
        }
    } else {
        croak "No discovery doc at $uri - $status_number $status_phrase";
    }
    $self;
}

sub new_request
{
    my $self = shift;
    local $ENV{'PERL_LWP_SSL_VERIFY_HOSTNAME'};
    if ($self->{'noVerifyHostname'} || ($host =~ /\.local$/)) {
        $ENV{'PERL_LWP_SSL_VERIFY_HOSTNAME'} = 0;
    }
    Arvados::Request->new();
}

sub load_config_file ($)
{
    my $config_file = shift;
    my %config;

    if (open (CONF, $config_file)) {
	while (<CONF>) {
	    next if /^\s*#/ || /^\s*$/;  # skip comments and blank lines
	    chomp;
	    my ($key, $val) = split /\s*=\s*/, $_, 2;
	    $config{$key} = $val;
	}
    }
    close CONF;
    return \%config;
}

1;
