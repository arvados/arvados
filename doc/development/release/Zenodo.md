[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Updating Zenodo Version of Arvados after Release

1.  Download a `.zip` of your new Arvados release from [GitHub Releases](https://github.com/arvados/arvados/tags)
2.  Log in to [Zenodo](https://zenodo.org/) using the credentials from `gopass "curii-systems/zenodo.org/sysadmin+zenodo@curii.com"`
3.  Go to the [Arvados record](https://zenodo.org/records/15213491) and press the the New Version button  
    (Using new versions lets us use the overarching DOI for our `citations.md` and keep all the versions together on Zenodo)
4.  In the form, update the following:
    1.  Upload the `.zip` file for this release you downloaded earlier
    2.  Request a new DOI for this version
    3.  Update the Publication Date for this release
    4.  Add any Creators who have worked on Arvados and aren’t listed
    5.  Under Additional Description, edit the links for Release Notes and (if you’re doing a major release) Documentation
    6.  Update the Version number with this release

Once you add a new version, you can’t change its DOI but everything else is editable if you accidentally make a mistake. So, don’t worry :) just edit the new version to fix it.
