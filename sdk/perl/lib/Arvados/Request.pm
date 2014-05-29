package Arvados::Request;
use Data::Dumper;
use LWP::UserAgent;
use URI::Escape;
use Encode;
use strict;
@Arvados::HTTP::ISA = qw(LWP::UserAgent);

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
    $self->{'ua'} = new LWP::UserAgent(@_);
    $self->{'ua'}->agent ("libarvados-perl/".$Arvados::VERSION);
    $self;
}

sub set_uri
{
    my $self = shift;
    $self->{'uri'} = shift;
}

sub process_request
{
    my $self = shift;
    my %req;
    my %content;
    my $method = $self->{'method'};
    if ($method eq 'GET' || $method eq 'HEAD') {
        $content{'_method'} = $method;
        $method = 'POST';
    }
    $req{$method} = $self->{'uri'};
    $self->{'req'} = new HTTP::Request (%req);
    $self->{'req'}->header('Authorization' => ('OAuth2 ' . $self->{'authToken'})) if $self->{'authToken'};
    $self->{'req'}->header('Accept' => 'application/json');
    my ($p, $v);
    while (($p, $v) = each %{$self->{'queryParams'}}) {
        $content{$p} = (ref($v) eq "") ? $v : JSON::encode_json($v);
    }
    my $content;
    while (($p, $v) = each %content) {
        $content .= '&' unless $content eq '';
        $content .= uri_escape($p);
        $content .= '=';
        $content .= uri_escape($v);
    }
    $self->{'req'}->content_type("application/x-www-form-urlencoded; charset='utf8'");
    $self->{'req'}->content(Encode::encode('utf8', $content));
    $self->{'res'} = $self->{'ua'}->request ($self->{'req'});
}

sub get_status
{
    my $self = shift;
    return ($self->{'res'}->code(),
	    $self->{'res'}->message());
}

sub get_body
{
    my $self = shift;
    return $self->{'res'}->content;
}

sub set_method
{
    my $self = shift;
    $self->{'method'} = shift;
}

sub set_query_params
{
    my $self = shift;
    $self->{'queryParams'} = shift;
}

sub set_auth_token
{
    my $self = shift;
    $self->{'authToken'} = shift;
}

sub get_headers
{
    ""
}

1;
