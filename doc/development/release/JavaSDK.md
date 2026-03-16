[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Releasing Java SDK packages

The Java SDK is distributed on the Sonatype Central Repository. Here are the steps to release a new jar file:

1.  Build and upload package using https://ci.arvados.org/view/All/job/build-java-sdk
2.  Go to [Sonatype Publishing Settings](https://central.sonatype.com/publishing/deployments) and log in with the appropriate credentials (gopass oss.sonatype.org/curii)
3.  Make sure you’re on the “Deployments” tab.
4.  Find the jar that was just uploaded by Jenkins. Click the “Publish” button and wait for the process to finish.

See [documentation about the publishing API we use](https://central.sonatype.org/publish/publish-portal-ossrh-staging-api/).

## Getting the authentication token for Sonatype

[Log into Sonatype](https://central.sonatype.com/usertoken) and under the account menu select “User Tokens” to review and manage tokens. Our current Jenkins token is stored in gopass as `curii-systems/websites/oss.sonatype.org/jenkins`.

## `gradle.properties`

To upload to Sonatype, you need the token (see above) and a secret key. You must upload a GPG-signed package. All these parameters are set in `gradle.properties` which we keep as a Jenkins secret. Note that the property values after the equals sign should not be quoted. I’m not certain if spaces are allowed around the equals sign, but currently it works with no extra spaces.

    ossrhUsername=...
    ossrhPassword=...
    signing.keyId=... 
    signing.password= 
    signing.secretKeyRingFile=...-secret-key.gpg 
