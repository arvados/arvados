=head1 NAME

Arvados -- client library for Arvados services

=head1 SYNOPSIS

  use Arvados;
  $arv = Arvados->new()->build(apiHost => 'arvados.local');
  
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

Hostname of API discovery service. Default: C<arvados.local>

=item apiProtocolScheme

Protocol scheme. Default: C<ARVADOS_API_PROTOCOL_SCHEME> environment
variable, or C<https>

=item apiToken

Authorization token. Default: C<ARVADOS_API_TOKEN> environment variable

=item apiService

Default C<arvados>

=item apiVersion

Default C<v1>

=back

=cut

package Arvados;
use JSON;
use Data::Dumper;
use IO::Socket::SSL;
use Carp;
use Arvados::ResourceAccessor;
use Arvados::ResourceMethod;
use Arvados::ResourceProxy;
use Arvados::ResourceProxyList;
use Arvados::Request;

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
    $self->{'authToken'} ||= $ENV{'ARVADOS_API_TOKEN'};
    $self->{'apiHost'} ||= $ENV{'ARVADOS_API_HOST'};
    $self->{'apiProtocolScheme'} ||= $ENV{'ARVADOS_API_PROTOCOL_SCHEME'};

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
    if ($opts{'noVerifyHostname'} || ($host =~ /\.local$/)) {
        $ENV{'PERL_LWP_SSL_VERIFY_HOSTNAME'} = 0;
    }
    Arvados::Request->new();
}

1;
