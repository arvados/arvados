#!/bin/bash

make

for target in `find -maxdepth 1 -type d |grep -v generated`; do
  if [[ "$target" == "." ]]; then
    continue
  fi
  target=${target#./}
  echo $target
  cd $target
  docker build -t arvados/build:$target .
  cd ..
done


