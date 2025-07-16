#!/usr/bin/env bash
shopt -s nullglob

for f in *.mp4; do
  base="${f%.mp4}"
  echo "🔄 Converting \"$f\" → \"$base.gif\" (native resolution)…"

  # 1) generate a single-frame palette
  ffmpeg -v warning -i "$f" \
    -vf "fps=10,format=rgb24,palettegen" \
    -update 1 \
    -y "${base}_palette.png"

  # 2) create the GIF using that palette at native size
  ffmpeg -v warning -i "$f" -i "${base}_palette.png" \
    -filter_complex "[0:v]fps=10,format=rgb24[x];[x][1:v]paletteuse" \
    -loop 0 \
    -y "${base}.gif"

  rm "${base}_palette.png"
done
