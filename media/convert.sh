#!/usr/bin/env bash
shopt -s nullglob

for f in *.mp4; do
  name="${f%.mp4}"
  echo "Converting \"$f\" → \"$name.gif\" (original dimensions)…"

  # 1) generate a palette at original size
  ffmpeg -v warning -i "$f" \
    -vf "fps=10,palettegen" \
    -y "${name}_palette.png"

  # 2) create the GIF using that palette, at original size
  ffmpeg -v warning -i "$f" -i "${name}_palette.png" \
    -filter_complex "fps=10[p];[p][1:v]paletteuse" \
    -loop 0 \
    -y "${name}.gif"

  # clean up
  rm "${name}_palette.png"
done
