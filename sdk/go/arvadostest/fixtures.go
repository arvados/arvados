package arvadostest

// IDs of API server's test fixtures
const (
	SpectatorToken        = "zw2f4gwx8hw8cjre7yp6v1zylhrhn3m5gvjq73rtpwhmknrybu"
	ActiveToken           = "3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi"
	AdminToken            = "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h"
	AnonymousToken        = "4kg6k6lzmp9kj4cpkcoxie964cmvjahbt4fod9zru44k4jqdmi"
	DataManagerToken      = "320mkve8qkswstz7ff61glpk3mhgghmg67wmic7elw4z41pke1"
	FooCollection         = "zzzzz-4zz18-fy296fx3hot09f7"
	NonexistentCollection = "zzzzz-4zz18-totallynotexist"
	HelloWorldCollection  = "zzzzz-4zz18-4en62shvi99lxd4"
	FooBarDirCollection   = "zzzzz-4zz18-foonbarfilesdir"
	FooPdh                = "1f4b0bc7583c2a7f9102c395f4ffc5e3+45"
	HelloWorldPdh         = "55713e6a34081eb03609e7ad5fcad129+62"
)

// A valid manifest designed to test various edge cases and parsing
// requirements
const PathologicalManifest = ". acbd18db4cc2f85cedef654fccc4a4d8+3 37b51d194a7513e45b56f6524f2d51f2+3 73feffa4b7f6bb68e44cf984c85f6e88+3+Z+K@xyzzy acbd18db4cc2f85cedef654fccc4a4d8+3 0:0:zero@0 0:1:f 1:0:zero@1 1:4:ooba 4:0:zero@4 5:1:r 5:4:rbaz 9:0:zero@9\n" +
	"./overlapReverse acbd18db4cc2f85cedef654fccc4a4d8+3 acbd18db4cc2f85cedef654fccc4a4d8+3 5:1:o 4:2:oo 2:4:ofoo\n" +
	"./segmented acbd18db4cc2f85cedef654fccc4a4d8+3 37b51d194a7513e45b56f6524f2d51f2+3 0:1:frob 5:1:frob 1:1:frob 1:2:oof 0:1:oof 5:0:frob 3:1:frob\n" +
	`./foo\040b\141r acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:baz` + "\n" +
	`./foo\040b\141r acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:b\141z\040w\141z` + "\n" +
	"./foo acbd18db4cc2f85cedef654fccc4a4d8+3 0:0:zero 0:3:foo\n" +
	". acbd18db4cc2f85cedef654fccc4a4d8+3 0:0:foo/zero 0:3:foo/foo\n"

// An MD5 collision.
var (
	MD5CollisionData = [][]byte{
		[]byte("\x0e0eaU\x9a\xa7\x87\xd0\x0b\xc6\xf7\x0b\xbd\xfe4\x04\xcf\x03e\x9epO\x854\xc0\x0f\xfbe\x9cL\x87@\xcc\x94/\xeb-\xa1\x15\xa3\xf4\x15\\\xbb\x86\x07Is\x86em}\x1f4\xa4 Y\xd7\x8fZ\x8d\xd1\xef"),
		[]byte("\x0e0eaU\x9a\xa7\x87\xd0\x0b\xc6\xf7\x0b\xbd\xfe4\x04\xcf\x03e\x9etO\x854\xc0\x0f\xfbe\x9cL\x87@\xcc\x94/\xeb-\xa1\x15\xa3\xf4\x15\xdc\xbb\x86\x07Is\x86em}\x1f4\xa4 Y\xd7\x8fZ\x8d\xd1\xef"),
	}
	MD5CollisionMD5 = "cee9a457e790cf20d4bdaa6d69f01e41"
)
