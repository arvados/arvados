package main

import (
	"flag"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type deprecatedOptions struct {
	flagSerializeIO     bool
	flagReadonly        bool
	neverDelete         bool
	signatureTTLSeconds int
}

var deprecated = deprecatedOptions{
	neverDelete:         !theConfig.EnableDelete,
	signatureTTLSeconds: int(theConfig.BlobSignatureTTL.Duration() / time.Second),
}

func (depr *deprecatedOptions) beforeFlagParse(cfg *Config) {
	flag.StringVar(&cfg.Listen, "listen", cfg.Listen, "see Listen configuration")
	flag.IntVar(&cfg.MaxBuffers, "max-buffers", cfg.MaxBuffers, "see MaxBuffers configuration")
	flag.IntVar(&cfg.MaxRequests, "max-requests", cfg.MaxRequests, "see MaxRequests configuration")
	flag.BoolVar(&depr.neverDelete, "never-delete", depr.neverDelete, "see EnableDelete configuration")
	flag.BoolVar(&cfg.RequireSignatures, "enforce-permissions", cfg.RequireSignatures, "see RequireSignatures configuration")
	flag.StringVar(&cfg.BlobSigningKeyFile, "permission-key-file", cfg.BlobSigningKeyFile, "see BlobSigningKey`File` configuration")
	flag.StringVar(&cfg.BlobSigningKeyFile, "blob-signing-key-file", cfg.BlobSigningKeyFile, "see BlobSigningKey`File` configuration")
	flag.StringVar(&cfg.SystemAuthTokenFile, "data-manager-token-file", cfg.SystemAuthTokenFile, "see SystemAuthToken`File` configuration")
	flag.IntVar(&depr.signatureTTLSeconds, "permission-ttl", depr.signatureTTLSeconds, "signature TTL in seconds; see BlobSignatureTTL configuration")
	flag.IntVar(&depr.signatureTTLSeconds, "blob-signature-ttl", depr.signatureTTLSeconds, "signature TTL in seconds; see BlobSignatureTTL configuration")
	flag.Var(&cfg.TrashLifetime, "trash-lifetime", "see TrashLifetime configuration")
	flag.BoolVar(&depr.flagSerializeIO, "serialize", depr.flagSerializeIO, "serialize read and write operations on the following volumes.")
	flag.BoolVar(&depr.flagReadonly, "readonly", depr.flagReadonly, "do not write, delete, or touch anything on the following volumes.")
	flag.StringVar(&cfg.PIDFile, "pid", cfg.PIDFile, "see `PIDFile` configuration")
	flag.Var(&cfg.TrashCheckInterval, "trash-check-interval", "see TrashCheckInterval configuration")
}

func (depr *deprecatedOptions) afterFlagParse(cfg *Config) {
	cfg.BlobSignatureTTL = arvados.Duration(depr.signatureTTLSeconds) * arvados.Duration(time.Second)
	cfg.EnableDelete = !depr.neverDelete
}
