#!/bin/sh

mkdir -p _site/sdk/python/arvados
cd _site/sdk/python/arvados
epydoc --html -o . "arvados"
