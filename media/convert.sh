#!/usr/bin/env bash
shopt -s nullglob

for f in *.mp4; do
  base="${f%.mp4}"
  echo "Converting \"$f\" → \"$base.gif\" (original resolution)…"

  # 1) Generate palette at the video’s native dimensions
  ffmpeg -v warning -i "$f" \
    -vf "fps=10,format=rgb8,palettegen" \
    -y "${base}_palette.png"

  # 2) Apply that palette—again at native size
  ffmpeg -v warning -i "$f" -i "${base}_palette.png" \
    -filter_complex "[0:v]fps=10,format=rgb8[x];[x][1:v]paletteuse" \
    -loop 0 \
    -y "${base}.gif"

  rm "${base}_palette.png"
done
