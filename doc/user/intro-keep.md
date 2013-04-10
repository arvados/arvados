---
layout: default
navsection: userguide
title: "Intro: Keep"
navorder: 3
---

# Intro: Keep

Keep is a content-addressable storage system. Its semantics are
inherently different from the POSIX-like file systems you're used to.

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

A data block contains between 1 byte and 64 MiB of data. Its locator
is the MD5 checksum of the data, followed by a plus sign and its size
in bytes (encoded as a decimal number).

`acbd18db4cc2f85cedef654fccc4a4d8+3`

Keep distributes data blocks among the available disks. It also stores
multiple copies of each block, so a single disk or node failure does
not cause any data to become unreachable.

### No "delete"

One of the side effects of the Keep write semantics is the lack of a
"delete" operation. Instead, Keep relies on garbage collection to
delete unneeded data blocks.

### Tagging valuable data

Valuable data must be marked explicitly by creating a Collection in
Arvados. Otherwise, the data blocks will be deleted during garbage
collection.

Use the arv(1) program to create a collection. For example:

    arv collections create --uuid "acbd18db4cc2f85cedef654fccc4a4d8+3"

## Getting started

Write three bytes of data to Keep.

    echo -n foo | whput -

Output:

    acbd18db4cc2f85cedef654fccc4a4d8+3

Retrieve the data.

    whget acbd18db4cc2f85cedef654fccc4a4d8+3

Output:

    foo

### Writing a collection

### Reading a file from a collection

### Adding a collection to Arvados

### Tagging a collection

### Mounting Keep as a read-only POSIX filesystem

### Mounting a single collection as a POSIX filesystem

