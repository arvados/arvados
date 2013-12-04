---
layout: default
navsection: userguide
title: "Intro: Keep"
navorder: 3
---

# Intro: Keep

Keep is a content-addressable storage system. Its semantics are
inherently different from the POSIX-like file systems you're used to.

> Explain what "content-addressable" means more specifically.
> Define "locator"

Using Keep looks like this:

1. Write data.
2. Receive locator.
3. Use locator to retrieve data.
4. Tag the locator with a symbolic name.

By contrast, using POSIX looks like this:

1. Choose locator (*i.e.*, filename).
2. Write data to locator.
3. Use locator to retrieve data.

Content addressing provides various benefits, including:

* Reduction of unnecessary data duplication
* Prevention of race conditions (a given locator always references the same data)
* Systematic client- and server-side verification of data integrity
* Provenance reporting (when combined with Arvados MapReduce jobs)

### Vocabulary

Keep arranges data into **collections** and **data blocks**.

A collection is analogous to a directory tree in a POSIX
filesystem. It contains subdirectories and filenames, and indicates
where to find the data blocks which comprise the files. It is encoded
in plain text.

> Can a collection contain sub-collections?
> The "plain text" encoding is JSON, right?  Either be specific or
> remove it because the user doesn't really need to know about the encoding
> at this level.

A data block contains between 1 byte and 64 MiB of data. Its locator
is the MD5 checksum of the data, followed by a plus sign and its size
in bytes (encoded as a decimal number).

`acbd18db4cc2f85cedef654fccc4a4d8+3`

> What does this locator encode?  Give an example so the astute
> reader could use "md5" herself to construct the id.

A locator may include additional "hints" to help the Keep store find a
data block more quickly. For example, in the locator
`acbd18db4cc2f85cedef654fccc4a4d8+3+K@{{ site.arvados_api_host }}` the
hint "+K@{{ site.arvados_api_host }}" indicates that the data block is
stored in the Keep instance called *{{ site.arvados_api_host }}*. The
hints are not necessary for Keep to locate a data block -- only the
MD5 hash is -- but they help improve performance.

Keep distributes data blocks among the available disks. It also stores
multiple copies of each block, so a single disk or node failure does
not cause any data to become unreachable.

### No "delete"

One of the side effects of the Keep write semantics is the lack of a
"delete" operation. Instead, Keep relies on garbage collection to
delete unneeded data blocks.

### Tagging valuable data

> Now this goes from background introduction to tutorial,
> so this should probably be split up

Valuable data must be marked explicitly by creating a Collection in
Arvados. Otherwise, the data blocks will be deleted during garbage
collection.

Use the arv(1) program to create a collection. For example:

    arv collection create --uuid "acbd18db4cc2f85cedef654fccc4a4d8+3"

> What does this actually do?

## Getting started

Write three bytes of data to Keep.

    echo -n foo | whput -

> What does "wh" stand for in the program name?

Output:

    acbd18db4cc2f85cedef654fccc4a4d8+3+K@arv01

> Explain that this is the locator that Keep has stored the data under

Retrieve the data.

    whget acbd18db4cc2f85cedef654fccc4a4d8+3+K@arv01

Output:

    foo


{% include alert-stub.html %}

### Writing a collection

### Reading a file from a collection

### Adding a collection to Arvados

### Tagging a collection

### Mounting Keep as a read-only POSIX filesystem

### Mounting a single collection as a POSIX filesystem

> Needs a yellow "this web page under construction" sign with a guy shoveling dirt.
