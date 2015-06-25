package arvadostest

const (
	SpectatorToken        = "zw2f4gwx8hw8cjre7yp6v1zylhrhn3m5gvjq73rtpwhmknrybu"
	ActiveToken           = "3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi"
	AnonymousToken        = "4kg6k6lzmp9kj4cpkcoxie964cmvjahbt4fod9zru44k4jqdmi"
	FooCollection         = "zzzzz-4zz18-fy296fx3hot09f7"
	NonexistentCollection = "zzzzz-4zz18-totallynotexist"
	HelloWorldCollection  = "zzzzz-4zz18-4en62shvi99lxd4"
	PathologicalManifest  = ". acbd18db4cc2f85cedef654fccc4a4d8+3 37b51d194a7513e45b56f6524f2d51f2+3 73feffa4b7f6bb68e44cf984c85f6e88+3+Z+K@xyzzy acbd18db4cc2f85cedef654fccc4a4d8+3 0:0:zero@0 0:1:f 1:0:zero@1 1:4:ooba 4:0:zero@4 5:1:r 5:4:rbaz 9:0:zero@9\n" +
		"./overlapReverse acbd18db4cc2f85cedef654fccc4a4d8+3 acbd18db4cc2f85cedef654fccc4a4d8+3 5:1:o 4:2:oo 2:4:ofoo\n" +
		"./segmented acbd18db4cc2f85cedef654fccc4a4d8+3 37b51d194a7513e45b56f6524f2d51f2+3 0:1:frob 5:1:frob 1:1:frob 1:2:oof 0:1:oof 5:0:frob 3:1:frob\n" +
		`./foo\040b\141r acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:baz` + "\n" +
		`./foo\040b\141r acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:b\141z\040w\141z` + "\n" +
		"./foo acbd18db4cc2f85cedef654fccc4a4d8+3 0:0:zero 0:3:foo\n" +
		". acbd18db4cc2f85cedef654fccc4a4d8+3 0:0:foo/zero 0:3:foo/foo\n"
)
