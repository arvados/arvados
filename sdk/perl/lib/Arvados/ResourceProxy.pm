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

sub resource_parameter_name
{
    my $self = shift;
    my $pname = $self->{'resourceAccessor'}->{'resourcesName'};
    $pname =~ s/s$//;           # XXX not a very good singularize()
    $pname;
}

1;
