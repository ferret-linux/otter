#!/bin/sh

for i do
  [ "$i" = --pty ] || set -- "$@" "$i"
  shift
done

/bin/su "$@"