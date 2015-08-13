#!/usr/bin/env perl

$ENV{ARVADOS_API_HOST_INSECURE} = 1;
use Authen::PAM qw(:constants);

for my $case (['good', 1, 'active', '3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi'],
              ['badtoken', 0, 'active', 'badtokenmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi'],
              ['badusername', 0, 'baduser', '3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi']) {
    dotest(@$case);
}
print "=== OK ===\n";

sub dotest {
    my ($label, $expect_ok, $user, $token) = @_;
    print "$label: ";
    my $service_name = 'login';
    $main::Token = $token;
    my $pamh = new Authen::PAM($service_name, $user, \&token_conv_func);
    ref($pamh) || die "Error code $pamh during PAM init!";
    $pamh->pam_set_item(PAM_RHOST(), '::1');
    $pamh->pam_set_item(PAM_RUSER(), 'none');
    $pamh->pam_set_item(PAM_TTY(), '/dev/null');
    my $flags = PAM_SILENT();
    $res = $pamh->pam_authenticate($flags);
    $msg = $pamh->pam_strerror($res);
    print "Result (code $res): $msg\n";
    if (($res == 0) != ($expect_ok == 1)) {
        die "*** FAIL ***\n";
    }
}

sub token_conv_func {
    my @res;
    while ( @_ ) {
        my $code = shift;
        my $msg = shift;
        my $ans;
        print "Message (type $code): $msg\n";
        if ($code == PAM_PROMPT_ECHO_OFF() || $code == PAM_PROMPT_ECHO_ON()) {
            $ans = $main::Token;
        }
        push @res, (0,$ans);
    }
    push @res, PAM_SUCCESS();
    return @res;
}
