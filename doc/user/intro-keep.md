---
layout: default
navsection: userguide
title: "Intro: Keep"
navorder: 3
---

# Using Keep

Keep is a content-addressable storage system. Its semantics are
inherently different from the POSIX-like file systems you're used to.

Using Keep looks like this:

1. Write data.
2. Receive locator.
3. Use locator to retrieve data.
4. Tag the locator with a symbolic name.

By contrast, POSIX works like this:

1. Choose locator (*i.e.*, filename).
2. Write data to locator.
3. Use locator to retrieve data.

Content addressing provides various benefits which we discuss
elsewhere.

### Getting started

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

