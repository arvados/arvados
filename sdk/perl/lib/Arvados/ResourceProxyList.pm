package Arvados::ResourceProxyList;

sub new
{
    my $class = shift;
    my $self = {};
    bless ($self, $class);
    $self->_init(@_);
}

sub _init
{
    my $self = shift;
    $self->{'serverResponse'} = shift;
    $self->{'resourceAccessor'} = shift;
    $self->{'items'} = [ map { Arvados::ResourceProxy->new($_, $self->{'resourceAccessor'}) } @{$self->{'serverResponse'}->{'items'}} ];
    $self;
}

1;
