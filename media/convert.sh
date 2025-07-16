#!/usr/bin/env bash
shopt -s nullglob

for f in *.mp4; do
  outfile="${f%.mp4}.webm"
  echo "Converting \"$f\" → \"$outfile\"…"
  ffmpeg -i "$f" \
    -c:v libvpx-vp9 -crf 30 -b:v 0 \
    -c:a libopus -b:a 64k \
    -metadata title="${outfile%.*}" \
    "$outfile"
done
