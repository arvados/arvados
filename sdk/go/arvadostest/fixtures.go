// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

// IDs of API server's test fixtures
const (
	SpectatorToken          = "zw2f4gwx8hw8cjre7yp6v1zylhrhn3m5gvjq73rtpwhmknrybu"
	ActiveToken             = "3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi"
	ActiveTokenUUID         = "zzzzz-gj3su-077z32aux8dg2s1"
	ActiveTokenV2           = "v2/zzzzz-gj3su-077z32aux8dg2s1/3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi"
	AdminUserUUID           = "zzzzz-tpzed-d9tiejq69daie8f"
	AdminToken              = "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h"
	AdminTokenUUID          = "zzzzz-gj3su-027z32aux8dg2s1"
	AnonymousToken          = "4kg6k6lzmp9kj4cpkcoxie964cmvjahbt4fod9zru44k4jqdmi"
	DataManagerToken        = "320mkve8qkswstz7ff61glpk3mhgghmg67wmic7elw4z41pke1"
	SystemRootToken         = "systemusertesttoken1234567890aoeuidhtnsqjkxbmwvzpy"
	ManagementToken         = "jg3ajndnq63sywcd50gbs5dskdc9ckkysb0nsqmfz08nwf17nl"
	ActiveUserUUID          = "zzzzz-tpzed-xurymjxw79nv3jz"
	FederatedActiveUserUUID = "zbbbb-tpzed-xurymjxw79nv3jz"
	SpectatorUserUUID       = "zzzzz-tpzed-l1s2piq4t4mps8r"
	UserAgreementCollection = "zzzzz-4zz18-uukreo9rbgwsujr" // user_agreement_in_anonymously_accessible_project
	FooCollectionName       = "zzzzz-4zz18-fy296fx3hot09f7 added sometime"
	FooCollection           = "zzzzz-4zz18-fy296fx3hot09f7"
	FooCollectionPDH        = "1f4b0bc7583c2a7f9102c395f4ffc5e3+45"
	NonexistentCollection   = "zzzzz-4zz18-totallynotexist"
	HelloWorldCollection    = "zzzzz-4zz18-4en62shvi99lxd4"
	FooBarDirCollection     = "zzzzz-4zz18-foonbarfilesdir"
	WazVersion1Collection   = "zzzzz-4zz18-25k12570yk1ver1"
	UserAgreementPDH        = "b519d9cb706a29fc7ea24dbea2f05851+93"
	HelloWorldPdh           = "55713e6a34081eb03609e7ad5fcad129+62"

	MultilevelCollection1 = "zzzzz-4zz18-pyw8yp9g3pr7irn"

	AProjectUUID    = "zzzzz-j7d0g-v955i6s2oi1cbso"
	ASubprojectUUID = "zzzzz-j7d0g-axqo7eu9pwvna1x"

	FooAndBarFilesInDirUUID = "zzzzz-4zz18-foonbarfilesdir"
	FooAndBarFilesInDirPDH  = "870369fc72738603c2fad16664e50e2d+58"

	Dispatch1Token    = "kwi8oowusvbutahacwk2geulqewy5oaqmpalczfna4b6bb0hfw"
	Dispatch1AuthUUID = "zzzzz-gj3su-k9dvestay1plssr"

	QueuedContainerRequestUUID = "zzzzz-xvhdp-cr4queuedcontnr"
	QueuedContainerUUID        = "zzzzz-dz642-queuedcontainer"

	RunningContainerUUID = "zzzzz-dz642-runningcontainr"

	CompletedContainerUUID         = "zzzzz-dz642-compltcontainer"
	CompletedContainerRequestUUID  = "zzzzz-xvhdp-cr4completedctr"
	CompletedContainerRequestUUID2 = "zzzzz-xvhdp-cr4completedcr2"

	CompletedDiagnosticsContainerRequest1UUID     = "zzzzz-xvhdp-diagnostics0001"
	CompletedDiagnosticsContainerRequest2UUID     = "zzzzz-xvhdp-diagnostics0002"
	CompletedDiagnosticsContainer1UUID            = "zzzzz-dz642-diagcompreq0001"
	CompletedDiagnosticsContainer2UUID            = "zzzzz-dz642-diagcompreq0002"
	DiagnosticsContainerRequest1LogCollectionUUID = "zzzzz-4zz18-diagcompreqlog1"
	DiagnosticsContainerRequest2LogCollectionUUID = "zzzzz-4zz18-diagcompreqlog2"

	CompletedDiagnosticsHasher1ContainerRequestUUID = "zzzzz-xvhdp-diag1hasher0001"
	CompletedDiagnosticsHasher2ContainerRequestUUID = "zzzzz-xvhdp-diag1hasher0002"
	CompletedDiagnosticsHasher3ContainerRequestUUID = "zzzzz-xvhdp-diag1hasher0003"
	CompletedDiagnosticsHasher1ContainerUUID        = "zzzzz-dz642-diagcomphasher1"
	CompletedDiagnosticsHasher2ContainerUUID        = "zzzzz-dz642-diagcomphasher2"
	CompletedDiagnosticsHasher3ContainerUUID        = "zzzzz-dz642-diagcomphasher3"

	Hasher1LogCollectionUUID = "zzzzz-4zz18-dlogcollhash001"
	Hasher2LogCollectionUUID = "zzzzz-4zz18-dlogcollhash002"
	Hasher3LogCollectionUUID = "zzzzz-4zz18-dlogcollhash003"

	ArvadosRepoUUID = "zzzzz-s0uqq-arvadosrepo0123"
	ArvadosRepoName = "arvados"
	FooRepoUUID     = "zzzzz-s0uqq-382brsig8rp3666"
	FooRepoName     = "active/foo"
	Repository2UUID = "zzzzz-s0uqq-382brsig8rp3667"
	Repository2Name = "active/foo2"

	FooCollectionSharingTokenUUID = "zzzzz-gj3su-gf02tdm4g1z3e3u"
	FooCollectionSharingToken     = "iknqgmunrhgsyfok8uzjlwun9iscwm3xacmzmg65fa1j1lpdss"

	WorkflowWithDefinitionYAMLUUID = "zzzzz-7fd4e-validworkfloyml"

	CollectionReplicationDesired2Confirmed2UUID = "zzzzz-4zz18-434zv1tnnf2rygp"

	ActiveUserCanReadAllUsersLinkUUID = "zzzzz-o0j2j-ctbysaduejxfrs5"

	TrustedWorkbenchAPIClientUUID = "zzzzz-ozdt8-teyxzyd8qllg11h"

	AdminAuthorizedKeysUUID = "zzzzz-fngyi-12nc9ov4osp8nae"

	CrunchstatForRunningJobLogUUID = "zzzzz-57u5n-tmymyrojrbtnxh1"

	IdleNodeUUID = "zzzzz-7ekkf-2z3mc76g2q73aio"

	TestVMUUID = "zzzzz-2x53u-382brsig8rp3064"

	CollectionWithUniqueWordsUUID = "zzzzz-4zz18-mnt690klmb51aud"

	LogCollectionUUID  = "zzzzz-4zz18-logcollection01"
	LogCollectionUUID2 = "zzzzz-4zz18-logcollection02"
)

// PathologicalManifest : A valid manifest designed to test
// various edge cases and parsing requirements
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

// BlobSigningKey used by the test servers
const BlobSigningKey = "zfhgfenhffzltr9dixws36j1yhksjoll2grmku38mi7yxd66h5j4q9w4jzanezacp8s6q0ro3hxakfye02152hncy6zml2ed0uc"
