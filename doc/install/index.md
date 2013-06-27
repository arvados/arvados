---
layout: default
navsection: installguide
title: Overview
navorder: 0
---

{% include alert-stub.html %}

# Installation Overview

1. Set up a cluster, or use Amazon
1. Create and mount Keep volumes
1. [Install the Arvados REST API server](install-api-server.html)
1. [Install the Arvados workbench application](install-workbench-app.html)
1. [Install the Crunch dispatcher](install-crunch-dispatch.html)
1. Create a Group named "Arvados Tutorials", owned by the system user. Create Links (link_class "resources", name "wants") from the tutorials group to sample data collections. Edit <code>page_content</code>, <code>page_title</code>, and <code>page_subtitle</code> properties to suit. These will be listed in the Tutorials section of your users' home pages.
