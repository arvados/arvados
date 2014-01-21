#!/bin/sh

mkdir -p _site/sdk/python
cd _site/sdk/python
epydoc --html -o . "arvados"
